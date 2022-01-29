package main

import (
	"flag"
	"fmt"
	"github.com/haochen233/socks5"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"
	"webFiveSniProxy/handlers"
)

type Config struct {
	Server struct {
		HttpHost  string `yaml:"httpHost"`
		HttpPort  string `yaml:"httpPort"`
		HttpsHost string `yaml:"httpsHost"`
		HttpsPort string `yaml:"httpsPort"`
	} `yaml:"server"`
	Proxy struct {
		Socks5Host string `yaml:"socks5Host"`
		Socks5Port string `yaml:"socks5Port"`
	} `yaml:"proxy"`
}

var (
	config     Config
	configFile *string
	//createConfigFile *bool
	socks5Client socks5.Client
)

func readConfig() {
	data, err := ioutil.ReadFile((*configFile) + ".yml")
	if err != nil {
		log.Fatal("Config file read error.")
	}
	fmt.Println(string(data))

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("Config unmarshal error.")
	}
	fmt.Println(config)
}

func init() {
	configFile = flag.String("C", "config", "Config File Path")
	//createConfigFile = flag.Bool("CCF", false, "Create Specified Config File With Default Values ? (Not Working)")
}

func main() {
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(-1)
	}

	readConfig()

	socks5Client = socks5.Client{
		ProxyAddr: net.JoinHostPort(config.Proxy.Socks5Host, config.Proxy.Socks5Port),
		Auth: map[socks5.METHOD]socks5.Authenticator{
			socks5.NO_AUTHENTICATION_REQUIRED: &socks5.NoAuth{},
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Http listener
	go func() {
		l, err := net.Listen("tcp", net.JoinHostPort(config.Server.HttpHost, config.Server.HttpPort))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Http Server Listening on: %v:%v", config.Server.HttpHost, config.Server.HttpPort)
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Print(err)
				continue
			}
			go handleConnection(conn, false)
		}

		wg.Done()
	}()

	// Https listener
	go func() {
		l, err := net.Listen("tcp", net.JoinHostPort(config.Server.HttpsHost, config.Server.HttpsPort))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Https Server Listening on: %v:%v", config.Server.HttpsHost, config.Server.HttpsPort)
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Print(err)
				continue
			}
			go handleConnection(conn, true)
		}

		wg.Done()
	}()

	wg.Wait()
}

func peekServerName(clientConn net.Conn, isHttps bool) (string, io.Reader, error) {
	if isHttps {
		log.Print("https client...")
		return handlers.PeekClientHello(clientConn)
	}

	log.Print("http client...")
	return handlers.PeekHttpReq(clientConn)
}

func handleConnection(clientConn net.Conn, isHttps bool) {
	defer func(clientConn net.Conn) {
		err := clientConn.Close()
		if err != nil {

		}
	}(clientConn)

	if err := clientConn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Print(err)
		return
	}

	serverName, clientReader, err := peekServerName(clientConn, isHttps)

	if err != nil {
		log.Print(err)
		return
	}

	if err := clientConn.SetReadDeadline(time.Time{}); err != nil {
		log.Print(err)
		return
	}

	log.Print("SNI: " + serverName)

	var dstPort string

	if isHttps {
		dstPort = config.Server.HttpsPort
	} else {
		dstPort = config.Server.HttpPort
	}

	backendConn, err := socks5Client.Connect(socks5.Version5, net.JoinHostPort(serverName, dstPort))

	if err != nil {
		log.Print(err)
		return
	}
	defer func(backendConn *net.TCPConn) {
		err := backendConn.Close()
		if err != nil {

		}
	}(backendConn)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		_, err := io.Copy(clientConn, backendConn)
		if err != nil {
			log.Print(err)
			return
		}
		err = clientConn.(*net.TCPConn).CloseWrite()
		if err != nil {
			log.Print(err)
			return
		}
		wg.Done()
	}()
	go func() {
		_, err := io.Copy(backendConn, clientReader)
		if err != nil {
			log.Print(err)
			return
		}
		err = backendConn.CloseWrite()
		if err != nil {
			log.Print(err)
			return
		}
		wg.Done()
	}()

	wg.Wait()
}
