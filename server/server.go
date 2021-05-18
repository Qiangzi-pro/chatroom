package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"
)

var (
	// todo 验证一下缓存型channel 使用pointer 和 value的区别
	messageChannel  = make(chan *Message, 8)
	enteringChannel = make(chan *User)
	leavingChannel  = make(chan *User)

	total = 0
)

type User struct {
	ID             int
	Addr           string
	EnterAt        time.Time
	MessageChannel chan string
}

type Message struct {
	OwnerID int
	Content string
}

func (u *User) String() string {
	return strconv.Itoa(u.ID) + ", " + u.Addr
}

func main() {
	listener, err := net.Listen("tcp", ":2020")
	if err != nil {
		panic(err)
	}

	go broadcaster()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept err: %v", err)
			continue
		}
		go handleConn(conn)
	}
}

func broadcaster() {
	users := make(map[*User]struct{})
	for {
		select {
		case user := <-enteringChannel:
			users[user] = struct{}{}
		case user := <-leavingChannel:
			delete(users, user)
			// 避免goroutine leak
			close(user.MessageChannel)
		case msg := <-messageChannel:
			for user := range users {
				if user.ID == msg.OwnerID {
					continue
				}
				user.MessageChannel <- msg.Content
			}
		}
	}

}

func handleConn(conn net.Conn) {
	defer conn.Close()

	user := &User{
		ID:             GenUserID(),
		Addr:           conn.RemoteAddr().String(),
		EnterAt:        time.Now(),
		MessageChannel: make(chan string, 8),
	}

	go sendMessage(conn, user.MessageChannel)

	user.MessageChannel <- fmt.Sprintf("welcome user: %s", user.String())
	messageChannel <- &Message{OwnerID: user.ID, Content: fmt.Sprintf("user: %d has enter", user.ID)}
	enteringChannel <- user

	var userActive = make(chan struct{})

	go func() {
		d := time.Minute * 5
		t := time.NewTicker(d)

		for {
			select {
			case <-userActive:
				t.Reset(d)
			case <-t.C:
				conn.Close()
				return
			}
		}
	}()

	// 不太好的实现，例如错误不好处理，消息传递了两次
	//clientCh := make(chan string)
	//go receiveMessage(conn, clientCh)
	//
	//ticker := time.NewTicker(d)
	//isExpired := false
	//for !isExpired {
	//	select {
	//	case msg := <-clientCh:
	//		messageChannel <- &Message{OwnerID: user.ID, Content: fmt.Sprintf("%d: %s", user.ID, msg)}
	//		ticker.Reset(d)
	//	case <-ticker.C:
	//		isExpired = true
	//	}
	//}

	input := bufio.NewScanner(conn)
	for input.Scan() {
		messageChannel <- &Message{OwnerID: user.ID, Content: fmt.Sprintf("%d: %s", user.ID, input.Text())}

		// 用户活跃
		userActive <- struct{}{}
	}

	if err := input.Err(); err != nil {
		log.Println("read err: ", err)
	}

	leavingChannel <- user
	messageChannel <- &Message{OwnerID: user.ID, Content: fmt.Sprintf("user: %d has left", user.ID)}
}

func receiveMessage(r net.Conn, ch chan<- string) {
	input := bufio.NewScanner(r)
	//input.Split()
	for input.Scan() {
		ch <- input.Text()
	}

	if err := input.Err(); err != nil {
		log.Println("read err: ", err)
	}
}

func sendMessage(conn io.Writer, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}

func GenUserID() int {
	total += 1
	return total
}
