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
		tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(config.Server.HttpHost, config.Server.HttpPort))
		if err != nil {
			log.Fatal(err)
		}

		l, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Http Server Listening on: %v:%v", config.Server.HttpHost, config.Server.HttpPort)

		for {
			conn, err := l.AcceptTCP()
			if err != nil {
				log.Print(err)
				time.Sleep(time.Second)
				continue
			}
			go handleConnection(conn, false)
		}

		wg.Done()
	}()

	// Https listener
	go func() {
		tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(config.Server.HttpsHost, config.Server.HttpsPort))
		if err != nil {
			log.Fatal(err)
		}

		l, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Https Server Listening on: %v:%v", config.Server.HttpsHost, config.Server.HttpsPort)

		for {
			conn, err := l.AcceptTCP()
			if err != nil {
				log.Print(err)
				time.Sleep(time.Second)
				continue
			}
			go handleConnection(conn, true)
		}

		wg.Done()
	}()

	wg.Wait()
}

func peekServerName(clientConn *net.TCPConn, isHttps bool) (string, io.Reader, error) {
	if isHttps {
		log.Print("https client...")
		return handlers.PeekClientHello(clientConn)
	}

	log.Print("http client...")
	return handlers.PeekHttpReq(clientConn)
}

func handleConnection(clientConn *net.TCPConn, isHttps bool) {
	serverName, clientReader, err := peekServerName(clientConn, isHttps)

	if err != nil {
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

	defer func(backendConn *net.TCPConn, clientConn net.Conn) {
		err := backendConn.Close()
		if err != nil {
			log.Print(err)
		}
		err = clientConn.Close()
		if err != nil {
			log.Print(err)
		}
		log.Print("Connection closed: " + serverName)
	}(backendConn, clientConn)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_, err := io.Copy(clientConn, backendConn)
		if err != nil {
			log.Print(err)
		}

		err = backendConn.CloseRead()
		if err != nil {
			log.Print(err)
		}

		err = clientConn.CloseWrite()
		if err != nil {
			log.Print(err)
		}
	}()
	go func() {
		for {
			err := clientConn.SetReadDeadline(time.Now().Add(30 * time.Second))

			if err == nil {
				writtenBytes, err := io.Copy(backendConn, clientReader)
				if err != nil {
					log.Print(err)
				}
				if writtenBytes == 0 {
					err = clientConn.CloseRead()
					if err != nil {
						log.Print(err)
					}

					err := backendConn.CloseWrite()
					if err != nil {
						log.Print(err)
					}

					break
				}
			} else {
				log.Print(err)
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()
}
