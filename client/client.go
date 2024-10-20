package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:3000") // conexão tcp
	if err != nil {
		log.Fatalf("Erro ao conectar ao servidor: %v", err)
	}
	defer conn.Close() // somente vai ser chamado quando a função terminar

	fmt.Println("Conectado ao servidor!")

	done := make(chan struct{})

	go func() {
		io.Copy(os.Stdout, conn) // tudo que digitar vai mandar pra conexão
		log.Println("Conexão com o servidor encerrada.")
		done <- struct{}{}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin) // scanner do input
		for scanner.Scan() {
			fmt.Fprintln(conn, scanner.Text()) // manda o texto do input pra conexão
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Erro ao ler entrada: %v", err)
		}
		conn.Close()
	}()

	<-done
}
