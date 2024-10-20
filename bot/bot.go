package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

func reverse(str string) string {
	reversed_str := ""
	for _, v := range str {
		reversed_str = string(v) + reversed_str
	}
	return reversed_str
}

func main() {
	conn, err := net.Dial("tcp", "localhost:3001") // conexão tcp
	if err != nil {
		log.Fatalf("Erro ao conectar ao servidor: %v", err)
	}
	defer conn.Close() // somente vai ser chamado quando a função terminar

	fmt.Println("Bot conectado ao servidor!")
	done := make(chan struct{})

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			message := strings.Fields(scanner.Text())

			text := "Mensagem de " + message[0] + ": " + message[4]
			fmt.Println(text)

			comand := "\\msg " + message[0] + " " + reverse(message[4])
			fmt.Fprintln(conn, comand)
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Erro ao ler entrada: %v", err)
		}
		conn.Close()
		done <- struct{}{}
	}()

	<-done
}
