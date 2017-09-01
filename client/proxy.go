package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"../lib"
	"../pb"

	"golang.org/x/net/context"
)

const leakyBufSize = 4108 // data.len(2) + hmacsha1(10) + data(4096)
const maxNBuf = 2048

var leakyBuf = lib.NewLeakyBuf(maxNBuf, leakyBufSize)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	if debug {
		log.Printf("socks connect from %q\n", conn.RemoteAddr().String())
	}

	cmd, err := lib.Handshake(conn)
	if err != nil {
		log.Printf("socks handshake err: %s", err)
		return
	}

	switch cmd {
	case lib.CmdConnect:
		tcpHandler(conn)
	case lib.CmdUDPAssociate:
		udpHandler(conn)
	default:
		return
	}
}

func tcpHandler(conn net.Conn) {
	addr, err := lib.GetReqAddr(conn)
	if err != nil {
		log.Printf("get request err: %s\n", err)
		return
	}

	addrStr := addr.String()
	if debug {
		log.Printf("target addr tcp %q\n", addrStr)
	}

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
		log.Println(err)
		return
	}

	stream, err := client.Pipeline(context.Background())
	if err != nil {
		log.Printf("establish stream err: %s\n", err)
		return
	}
	defer func() {
		err = stream.CloseSend()
		if err != nil {
			log.Printf("close stream err: %s\n", err)
		}
	}()

	go func() {
		for {
			p, err := stream.Recv()
			if err == io.EOF {
				break
			}

			if err != nil {
				log.Printf("stream recv err: %s\n", err)
				break
			}

			_, err = conn.Write(p.Data)
			if err != nil {
				log.Printf("conn write err: %s\n", err)
				break
			}
		}
	}()

	frame := &pb.Payload{Data: []byte(addrStr)}

	err = stream.Send(frame)
	if err != nil {
		log.Printf("first frame send err: %s\n", err)
		return
	}

	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)

	for {
		// set read timeout
		n, err := conn.Read(buff)

		if n > 0 {
			frame.Data = buff[:n]
			err = stream.Send(frame)
			if err != nil {
				log.Printf("stream send err: %s\n", err)
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

	if debug {
		log.Printf("closed tcp connection to %s\n", addr)
	}
}

func udpHandler(conn net.Conn) {
	// do not using client indicate add
	_, err := lib.GetReqAddr(conn)
	if err != nil {
		log.Printf("get request err: %s\n", err)
		return
	}

	udpLn, err := net.ListenPacket("udp", "")
	if err != nil {
		log.Printf("create udp conn err: %s\n", err)
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
		log.Println(err)
		return
	}

	stream, err := client.PipelineUDP(context.Background())
	if err != nil {
		log.Printf("establish stream err: %s\n", err)
		return
	}
	defer func() {
		err = stream.CloseSend()
		if err != nil {
			log.Printf("close stream err: %s\n", err)
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
				log.Printf("stream recv err: %s\n", err)
				break
			}

			_, err = udpLn.WriteTo(p.Data, netInfo.BNDAddr)
			if err != nil {
				log.Printf("conn write err: %s\n", err)
				break
			}

			if debug {
				log.Printf("udp %q <-- %q\n", netInfo.BNDAddr.String(), netInfo.DSTAddr)
			}
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

				if debug {
					log.Printf("udp %q --> %q\n", netInfo.BNDAddr.String(), netInfo.DSTAddr)
				}

				if !first {
					first = true
					err := stream.Send(&pb.Payload{Data: []byte(netInfo.DSTAddr)})
					if err != nil {
						log.Printf("first frame send err: %s\n", err)
						return
					}
				}

				data := buff[3+len(dst): n]

				err = stream.Send(&pb.Payload{Data: data})
				if err != nil {
					log.Printf("stream send err: %s\n", err)
					return
				}
			}(buff)

		}

		if err != nil {
			break
		}
	}

	if debug {
		log.Printf("closed udp connection to %s\n", netInfo.DSTAddr)
	}
}
