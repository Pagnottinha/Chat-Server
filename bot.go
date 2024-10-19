package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
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
		io.Copy(os.Stdout, conn) // tudo que digitar vai mandar pra conexão
		log.Println("Conexão com o servidor encerrada.")
		done <- struct{}{}
	}()

	go func() {
		scanner := bufio.NewScanner(conn) 
		for scanner.Scan() {
			message := strings.Fields(scanner.Text())	
			comand := "\\msg " + message[0] + " " + reverse(message[4]) + "\n"
        	io.WriteString(conn, comand)
		}
		if err := scanner.Err(); err != nil {
			log.Println("Erro ao ler entrada: %v", err)
		}
		conn.Close()
	}()

	<-done
}
