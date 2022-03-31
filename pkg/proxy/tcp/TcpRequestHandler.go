package tcp

import (
	"bufio"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"strconv"
	"strings"
)

func StartMysqlProxy(port int) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal().Msgf("err: %v", err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Error().Msgf("accept err:%v", err)
			continue
		}

		//连接到mysql
		dial, err := net.Dial("tcp", "10.30.30.78:23306")
		if err != nil {
			log.Fatal().Msgf("连接到mysql服务器异常 %v", err)
		}

		go io.Copy(dial, conn)
		go io.Copy(conn, dial)

	}
}
func StartTcp(port int) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal().Msgf("err: %v", err)
	}
	println("tcp 服务启动，端口", port)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Error().Msgf("accept err:%v", err)
			continue
		}

		target := getTargetConn(conn)
		if target == nil {
			conn.Write([]byte("连接异常"))
			continue
		}
		go io.Copy(conn, target)
		go io.Copy(target, conn)

	}
}

func getTargetConn(conn net.Conn) net.Conn {
	data := make([]byte, 1024)
	bufConn := bufio.NewReader(conn)
	n, err := bufConn.Read(data)
	if err != nil {
		log.Error().Msgf("read from conn err:%v", err)
		return nil
	}
	data = data[:n]
	println("data：\n", string(data), data)
	var dial net.Conn
	if isRedis(data) {
		if dial, err = net.Dial("tcp", "10.30.30.78:26379"); err != nil {
			log.Error().Msgf("redis连接失败,error %v", err)
			return nil
		}
	} else if isMongoDB(data) {
		log.Info().Msgf("是mongoDB协议")
		if dial, err = net.Dial("tcp", "10.30.30.106:27017"); err != nil {
			log.Error().Msgf("mongoDB连接失败,error %v", err)
			return nil
		}
	}

	if dial != nil {
		dial.Write(data)
	}

	return dial
}

func isMongoDB(data []byte) bool {
	return strings.ContainsAny(string(data), "mongo")
}

func isRedis(data []byte) bool {
	if data[0] == 42 {
		size, err := strconv.Atoi(string(data[1]))
		if err != nil {
			log.Warn().Msgf("不是redis协议,error : %v", err)
			return false
		}
		counter := getCounter(data)
		if size == (counter-1)/2 {
			log.Info().Msgf("是redis协议")
			return true
		}

	}
	return false
}

//getCounter 统计换行符数量
func getCounter(data []byte) int {
	counter := 0
	for _, c := range data {
		if c == 10 || c == 0 {
			counter = counter + 1
		}
	}
	return counter
}

func serverConn(conn net.Conn) error {
	defer conn.Close()
	bufConn := bufio.NewReader(conn)

	//Read the version type
	version := []byte{0}
	if _, err := bufConn.Read(version); err != nil {
		if err.Error() == "EOF" {
			return nil
		}
		log.Error().Msgf("Failed to get version byte : %v", err)
		return err
	}
	return nil
}
