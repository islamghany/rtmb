package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/islamghany/rtmb/commands"
	"github.com/islamghany/rtmb/parser"
	"github.com/islamghany/rtmb/topic"
)

func main() {
	var (
		port      string
		tlsCert   string
		tlsKey    string
		useTLS    bool
		verbose   bool
		tlsConfig *tls.Config
	)

	flag.StringVar(&port, "port", "4222", "Port to listen on")
	flag.BoolVar(&useTLS, "tls", false, "Enable TLS encryption")
	flag.StringVar(&tlsCert, "tls-cert", "server.crt", "TLS certificate file")
	flag.StringVar(&tlsKey, "tls-key", "server.key", "TLS key file")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.Parse()

	if useTLS {
		var err error
		tlsConfig, err = loadTLSConfig(tlsCert, tlsKey)
		if err != nil {
			log.Fatalf("Failed to load TLS configuration: %v", err)
		}
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down server...")
		listener.Close()
		os.Exit(0)
	}()

	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	topicManager := topic.NewTopic()

	log.Printf("Server started, listening on port %s", port)

	// go topicManager.PrintTopic() // TODT remove this line

	for {
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				log.Printf("Temporary error accepting connection %v", netErr)
				continue
			}
			log.Printf("Fatal error accepting connection: %v", err)
			continue
		}
		if useTLS {
			tlsConn := tls.Server(conn, tlsConfig)
			go handleConnection(tlsConn, topicManager, port, verbose)
		} else {
			go handleConnection(conn, topicManager, port, verbose)
		}
	}
}

func handleConnection(conn net.Conn, topicManager *topic.Topic, port string, verbose bool) {

	defer func() {
		conn.Close()
		fmt.Printf("Connection from %v closed\n", conn.RemoteAddr())
	}()

	commander := commands.NewCommander(&commands.CommanderConfig{
		Conn: conn, Topic: topicManager, Port: port,
	})

	if verbose {
		log.Printf("New connection from %v", conn.RemoteAddr())
	}

	// Create a context to manage the lifecycle of the connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to manage concurrent goroutines
	var wg sync.WaitGroup

	// Channel to signal when the connection should be closed
	stopPing := make(chan struct{})
	// defer close(stopPing)

	if err := commander.SendInfo(); err != nil {
		log.Printf("Failed to send INFO: %v", err)
		return
	}

	// CONNECT Command provide more information about the current connection as well as security information.e.g verbose, tls_required, jwt, etc.
	// Wait for CONNECT command from client
	if !waitForConnect(conn) {
		log.Println("Connection closed: missing CONNECT command")
		return
	}

	// PING/PONG implement a simple keep-alive mechanism to ensure the client is still connected
	// it will send a PING to the client every amount of time and expect a PONG in return within a certain time frame
	// if the client does not respond with PONG, the connection will be closed
	wg.Add(1)
	go managePingPong(ctx, &wg, conn, stopPing)
	defer close(stopPing)

	reader := bufio.NewReader(conn)
	// Listen for incoming commands from the client
	for {
		// Set read deadline to prevent hanging connections
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Minute)); err != nil {
			log.Printf("Error setting read deadline for %v: %v", conn.RemoteAddr(), err)
			break
		}

		cmd, err := parser.Parse(reader)
		if err != nil {
			proceed := handleParseError(conn, err)
			if !proceed {
				break
			}
			continue
		}

		// Reset read deadline after successful read
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			log.Printf("Error resetting read deadline for %v: %v", conn.RemoteAddr(), err)
			break
		}

		// Handle received commands
		if err := commander.HandleCommand(cmd); err != nil {
			log.Printf("Error handling command: %v", err)
		}

		// Stop the ping routine if the client responded with PONG
		if cmd.Name == parser.PONG {
			select {
			case stopPing <- struct{}{}:
			default:
				// Do nothing if stopPing channel is not ready
			}
		}

	}
	// Cancel the context and wait for goroutines to complete
	cancel()
	wg.Wait()

	// Clean up client subscriptions on disconnect
	commander.CleanupSubscriptions()
}

func waitForConnect(conn net.Conn) bool {
	reader := bufio.NewReader(conn)

	// Set a short read deadline to wait for the CONNECT command
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
		return false
	}

	// Parse the next command from the client
	cmd, err := parser.Parse(reader)
	if err != nil {
		handleConnectError(err)
		return false
	}

	if cmd.Name != parser.CONNECT {
		log.Printf("Expected CONNECT, received %s from %v", cmd.Name, conn.RemoteAddr())
		return false
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("Error resetting read deadline after CONNECT: %v", err)
		return false
	}

	log.Printf("Received CONNECT from %v", conn.RemoteAddr())

	return true
}

func managePingPong(ctx context.Context, wg *sync.WaitGroup, conn net.Conn, stopPing <-chan struct{}) {
	defer wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	pingSent := false
	pongReceived := false
	for {
		select {
		case <-ctx.Done():
			return // Exit if the parent function is done
		case <-stopPing:
			pongReceived = true
			// Client responded with PONG, reset ticker and continue
			ticker.Reset(5 * time.Minute)
		case <-ticker.C:
			if pingSent && !pongReceived {
				log.Printf("Client %v did not respond to PING, closing connection", conn.RemoteAddr())
				conn.Close()
				return
			}
			// Send PING to the client
			if _, err := conn.Write([]byte("PING\r\n")); err != nil {
				log.Printf("Error writing PING to client: %v", err)
				return
			}
			pingSent = true
			pongReceived = false
			// Wait for PONG within a shorter timeframe
			ticker.Reset(20 * time.Second)
		}
	}
}

func handleParseError(conn net.Conn, err error) bool {
	if err == io.EOF {
		log.Printf("Client %v closed connection", conn.RemoteAddr())
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		log.Printf("Read timeout from client %v, closing connection", conn.RemoteAddr())
		return false
	}
	log.Printf("Parse error from client %v: %v", conn.RemoteAddr(), err)
	errStr := fmt.Sprintf("-ERR '%v'\r\n", err)
	if _, writeErr := conn.Write([]byte(errStr)); writeErr != nil {
		log.Printf("Error writing error message to client %v: %v", conn.RemoteAddr(), writeErr)
		return false
	}
	return true
}

func handleConnectError(err error) {
	if err == io.EOF {
		log.Println("Client closed connection before CONNECT")
	} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		log.Println("CONNECT timeout")
	} else {
		log.Printf("Error receiving CONNECT: %v", err)
	}
}

func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("could not load TLS key pair: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}, nil
}
