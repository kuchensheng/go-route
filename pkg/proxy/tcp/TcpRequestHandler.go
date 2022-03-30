package tcp

import (
	"bytes"
	"fmt"
	"github.com/armon/go-socks5"
	"github.com/rs/zerolog/log"
	"io"
	"net"
)

func StartTcp(port int) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal().Msgf("err: %v", err)
	}
	go func() {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal().Msgf("accept err:%v", err)
		}
		defer conn.Close()
		buf := make([]byte, 4)
		if _, err := io.ReadAtLeast(conn, buf, 4); err != nil {
			log.Fatal().Msgf("read err:%v", err)
		}
		if !bytes.Equal(buf, []byte("ping")) {
			log.Fatal().Msgf("bad err:%v", buf)
		}
		conn.Write([]byte("pong"))
	}()

	//Create a socks server
	conf := &socks5.Config{}

	serv, err := socks5.New(conf)
	if err != nil {
		log.Fatal().Msgf("socks5 server create fail,err : %v", err)
	}
	if err := serv.ListenAndServe("tcp", fmt.Sprintf(":%d", port)); err != nil {
		log.Fatal().Msgf("socks5 server listener err : %v", err)
	}

}
