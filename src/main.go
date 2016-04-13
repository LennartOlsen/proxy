package main

import (
	"net"
	"bufio"
	"net/http"
	"log"
	"io"
	"strconv"
	"time"
)

type Backend struct {
	net.Conn //Embedded type - Like now is backend inheriting from net.conn (its a child)
		 //So its a wrapper around this other type called net.Conn
	Reader *bufio.Reader
	Writer *bufio.Writer
}

var backendQueue chan *Backend

// init runs when the program starts
// kinda like main, but this one can be in every module of a program
// There can only be one main
func init() {
	backendQueue = make(chan *Backend, 10) 	//channel at a size of 10
						//This is a buffered channel.
						//This means that it can hold 10 Backends before it starts to block
}

func main() {
	log.Println("Listening")
	//listen for a connection on socket 8080 (always)
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	}
	//forever for
	for {
		//accept all connections
		if conn, err := ln.Accept(); err == nil {
			log.Println("got connection")
			//Mega concurrent
			// Take this function call and go run it somewhere else
			go handleConnection(conn)
		}
	}
}

func handleConnection(conn net.Conn){
	// Always run this, notmatter what!
	// This ensures that we always close our connection when we are done with it or if it fails
	defer conn.Close()

	// Buffered Io makes all faster :D
	// Remember to flush that baddie
	// Bufio reads from the socket it has been given (8080)
	reader := bufio.NewReader(conn)

	// Now that we have a connection pr user (not one pr instance)
	// we can read requests as in a loop pr user :D
	for{
		req, err := http.ReadRequest(reader);
		log.Println("Reading request");

		// As we need to know when the user goes away (the connection closes)
		// We need to return
		if err != nil {
			// although if the connection failed for some reason we want to log it
			// but still return at the end.
			if err != io.EOF {
				log.Printf("Failed to read request: %s", err)
			}
			// kills of this function ( see the defer? )
			return
		}


		be, err := getBackend()

		if err != nil {
			return
		}

		// Forward the request to the "backend"
		if err := req.Write(be.Writer); err == nil {
			log.Println("Writing request");
			be.Writer.Flush()

			//read the response from the backend
			resp, err := http.ReadResponse(be.Reader, req)
			if err != nil {
				log.Printf("Error on reading response %v", err)
				return;
			}

			log.Println("reading response")

			bytes := UpdateStats(req, resp);
			resp.Header.Set("X-Bytes", strconv.FormatInt(bytes, 10))

			//send response to the client.
			resp.Close = true
			if err := resp.Write(conn); err == nil {
				log.Printf("%s: %d", req.URL.Path, resp.StatusCode)
				log.Printf("Content Length: %d", resp.ContentLength)
			}

		}

		go queueBackend(be)

	}
}

func getBackend() (*Backend, error) {
	// kinda like a switch
	// but this bad boy blocks until one of these cases is possible
	select {
	// be := <-backendQueue is the syntax for reading from a channel
	case be := <-backendQueue :
		log.Println("has backend");
		return be, nil
	case <- time.After(100 * time.Millisecond) : 	//A way of waiting for 100ms before acting on anything. Ex:
							//A backend could become availible and go will select the other case
		log.Println("Creates backend");

		// Connect to the backend webserver
		// Consider resolving this somehow (map between port and service)
		be, err := net.Dial("tcp", "127.0.0.1:8081")

		log.Println("dials 127.0.0.1:8081")

		if err != nil {
			return nil, err
		}

		return &Backend{
			Conn: 	be,
			Reader: bufio.NewReader(be),
			Writer:	bufio.NewWriter(be),
		}, nil
	}
}

func queueBackend(be *Backend) {
	select {
	case backendQueue <- be :
		//Backend re-enqueued all is gooood
	case <- time.After(1 * time.Second) :
		be.Close() //if there is no space we shut it down
	}
}