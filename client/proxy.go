package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"time"

	"../lib"
	"../pb"
	"../log"

	"golang.org/x/net/context"
)

const leakyBufSize = 4108 // data.len(2) + hmacsha1(10) + data(4096)
const maxNBuf = 2048

var leakyBuf = lib.NewLeakyBuf(maxNBuf, leakyBufSize)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	cmd, err := lib.Handshake(conn)
	if err != nil {
		log.Errorf("socks handshake err: %s", err)
		return
	}

	switch cmd {
	case lib.CmdConnect:
		tcpHandler(conn)
	case lib.CmdUDPAssociate:
		udpHandler(conn)
	default:
		log.Errorf("socks cmd %v not supported", cmd)
		return
	}
}

func tcpHandler(conn net.Conn) {
	addr, err := lib.GetReqAddr(conn)
	if err != nil {
		log.Errorf("get req addr err: %s", err)
		return
	}

	addrStr := addr.String()

	log.Debugf("tcp conn  %q<-->%q", conn.RemoteAddr().String(), addrStr)

	// Sending connection established message immediately to client.
	// This cost some round trip time for creating socks connection with the client.
	// But if connection failed, the client will get connection reset error.
	//
	// Notice that the server response bind addr & port could be ignore by the socks5 client
	// 0x00 0x00 0x00 0x00 0x00 0x00 is meaning less for bind addr block.
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err != nil {
		return
	}

	client, err := gRPCClient()
	if err != nil {
		log.Errorln(err.Error())
		return
	}

	stream, err := client.Pipeline(context.Background())
	if err != nil {
		log.Errorf("establish stream err: %s", err)
		return
	}
	defer func() {
		stream.CloseSend()
	}()

	isClosed := false

	go func() {
		for {
			p, err := stream.Recv()

			if err != nil {
				if err == io.EOF {
					break
				}
				log.Errorf("stream recv err: %s", err)
				break
			}

			if isClosed {
				break
			}

			_, err = conn.Write(p.Data)
			if err != nil {
				log.Errorf("conn write err: %s", err)
				break
			}
		}

		if !isClosed {
			isClosed = true
		}
	}()

	frame := &pb.Payload{Data: []byte(addrStr)}

	err = stream.Send(frame)
	if err != nil {
		log.Errorf("first frame send err: %s", err)
		return
	}

	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)

	for {
		n, err := conn.Read(buff)

		if isClosed {
			break
		}

		if n > 0 {
			frame.Data = buff[:n]
			err = stream.Send(frame)
			if err != nil {
				log.Errorf("stream send err: %s", err)
				return
			}
		}

		if err != nil {
			// Always "use of closed network connection", but no easy way to
			// identify this specific error. So just leave the error along for now.
			// More info here: https://code.google.com/p/go/issues/detail?id=4373
			/*
				if bool(Debug) && err != io.EOF {
					Debug.Println("read:", err)
				}
			*/
			break
		}
	}

	isClosed = true

	log.Debugf("tcp close %q<-->%q", conn.RemoteAddr().String(), addrStr)
}

func udpHandler(conn net.Conn) {
	// do not using client indicate add
	_, err := lib.GetReqAddr(conn)
	if err != nil {
		log.Errorf("get request err: %s", err)
		return
	}

	udpLn, err := net.ListenPacket("udp", "")
	if err != nil {
		log.Errorf("create udp conn err: %s", err)
		// optional reply
		// 05 01 00 ... for generate ip field
		return
	}
	defer udpLn.Close()

	udpLn.SetReadDeadline(time.Now().Add(time.Second * 600))

	serverBindAddr, err := net.ResolveUDPAddr("udp", udpLn.LocalAddr().String())
	replay := []byte{0x05, 0x00, 0x00, 0x01} // header of server relpy association
	rawServerBindAddr := bytes.NewBuffer([]byte{0x0, 0x0, 0x0, 0x0})
	if err = binary.Write(rawServerBindAddr, binary.BigEndian, int16(serverBindAddr.Port)); err != nil {
		return
	}
	replay = append(replay, rawServerBindAddr.Bytes()[:6]...)
	if _, err = conn.Write(replay); err != nil {
		return
	}

	client, err := gRPCClient()
	if err != nil {
		log.Errorln(err)
		return
	}

	stream, err := client.PipelineUDP(context.Background())
	if err != nil {
		log.Errorf("establish stream err: %s", err)
		return
	}
	defer func() {
		err = stream.CloseSend()
		if err != nil {
			log.Errorf("close stream err: %s", err)
		}
	}()

	// natinfo keep the udp nat info for each socks5 association pair
	type natTableInfo struct {
		DSTAddr string
		BNDAddr net.Addr
	}

	var netInfo = natTableInfo{}

	go func() {
		for {
			p, err := stream.Recv()
			if err == io.EOF {
				break
			}

			if err != nil {
				log.Errorf("stream recv err: %s", err)
				break
			}

			_, err = udpLn.WriteTo(p.Data, netInfo.BNDAddr)
			if err != nil {
				log.Errorf("conn write err: %s", err)
				break
			}

			log.Debugf("udp %q <-- %q", netInfo.BNDAddr.String(), netInfo.DSTAddr)
		}
	}()

	buff := make([]byte, lib.UDPMaxSize) // TODO using pool is better
	first := false                       // TODO need pool to guarantee and first correct?
	for {
		n, addr, err := udpLn.ReadFrom(buff)

		if n > 0 {
			netInfo.BNDAddr = addr // TODO may be need cache add add time exp?

			go func(buff []byte) {
				// 0x00 0x00 for rsv
				// 0x00 for fragment

				/*
				      +----+------+------+----------+----------+----------+
				      |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
				      +----+------+------+----------+----------+----------+
				      | 2  |  1   |  1   | Variable |    2     | Variable |
				      +----+------+------+----------+----------+----------+
				 */

				dst := lib.SplitAddr(buff[3:n])

				netInfo.DSTAddr = dst.String()

				log.Debugf("udp %q --> %q", netInfo.BNDAddr.String(), netInfo.DSTAddr)

				if !first {
					first = true
					err := stream.Send(&pb.Payload{Data: []byte(netInfo.DSTAddr)})
					if err != nil {
						log.Errorf("first frame send err: %s", err)
						return
					}
				}

				data := buff[3+len(dst): n]

				err = stream.Send(&pb.Payload{Data: data})
				if err != nil {
					log.Errorf("stream send err: %s", err)
					return
				}
			}(buff)

		}

		if err != nil {
			break
		}
	}

	log.Debugf("closed udp connection to %s", netInfo.DSTAddr)
}
