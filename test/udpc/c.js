const Socks = require("socks");
const dgram = require("dgram");

let options = {
  proxy: {
    ipaddress: "127.0.0.1",
    port: 50050,
    type: 5,
    command: "associate" // Since we are using associate, we must specify it here.
  },
  target: {
    // When using associate, either set the ip and port to 0.0.0.0:0 or the expected source of incoming udp packets.
    // Note: Some SOCKS servers MAY block associate requests with 0.0.0.0:0 endpoints.
    // Note: ipv4, ipv6, and hostnames are supported here.
    host: "0.0.0.0",
    port: 0
  }
};

Socks.createConnection(options, function(err, socket, info) {
  if (err) console.log(err);
  else {
    // Associate request has completed.
    // info object contains the remote ip and udp port to send UDP packets to.
    console.log(info);
    // { port: 42803, host: '202.101.228.108' }

    console.log(dgram);

    let udp = new dgram.Socket("udp4");

    udp.on("close", () => {
      console.log("socket已关闭");
    });

    udp.on("error", err => {
      console.log(`server error:\n${err.stack}`);
      udp.close();
    });

    udp.on("message", (msg, rinfo) => {
      console.log(`server got: ${msg} from ${rinfo.address}:${rinfo.port}`);
    });

    udp.on("listening", () => {
      const address = udp.address();
      console.log(`server listening ${address.address}:${address.port}`);
    });

    // In this example we are going to send "Hello" to 1.2.3.4:2323 through the SOCKS proxy.

    //udp.bind(41234);

    setInterval(() => {
      let reqStr = "hello" + Math.random();
      let pack = Socks.createUDPFrame(
        { host: "1.2.3.4", port: 2323 },
        new Buffer(reqStr)
      );
      udp.send(pack, 0, pack.length, info.port, info.host);
      console.log("req", reqStr);
    }, 1000);
  }
});
