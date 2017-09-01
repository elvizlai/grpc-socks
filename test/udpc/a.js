var SocksClient = require('socks-client');

var options = {
   proxy: {
       ipaddress: "127.0.0.1", // Random public proxy 
       port: 50050,
       type: 5 // type is REQUIRED. Valid types: [4, 5]  (note 4 also works for 4a) 
   },
   target: {
       host: "google.com", // can be an ip address or domain (4a and 5 only) 
       port: 80
   },
   command: 'connect'  // This defaults to connect, so it's optional if you're not using BIND or Associate. 
};

SocksClient.createConnection(options, function(err, socket, info) {
   if (err)
       console.log(err);
   else {
       // Connection has been established, we can start sending data now: 
       socket.write("GET / HTTP/1.1\nHost: google.com\n\n");
       socket.on('data', function(data) {
           console.log(data.length);
           console.log(data);
       });

       // PLEASE NOTE: sockets need to be resumed before any data will come in or out as they are paused right before this callback is fired. 
       socket.resume();

       // 569 
       // <Buffer 48 54 54 50 2f 31 2e 31 20 33 30 31 20 4d 6f 76 65 64 20 50 65... 
   }
});