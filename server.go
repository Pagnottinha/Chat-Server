package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// client que conecta com o servidor
type client struct {
	nickname string
	ch       chan<- string // canal de mensagem
	bot      bool
}

var (
	clients            = make(map[string]*client)      // todos os clientes conectados: key string ->  value client
	entering           = make(chan *client)            // canal de entrada
	leaving            = make(chan *client)            // canal de saida
	broadcast_messages = make(chan message)            // canal para broadcasts
	private_massages   = make(chan message)            // canal para mensagens privadas
	commands           = make(chan command)            // canal para comandos
	availableCommands  = map[string]func(cmd command){ // hashtable de comandos
		"changenick": handleChangeNick,
		"msg":        handleMsg,
		"help":       handleHelp,
	}
)

type message struct {
	from    string
	to      string
	content string
}

type command struct {
	from    string
	command string
	args    []string
}

// gorrotina para canais de entrada e saida
func client_inout() {
	for {
		select {
		case cli := <-entering:
			clientType := "Usuário"
            if cli.bot {
                clientType = "Bot"
            }

			clients[cli.nickname] = cli
			broadcast_messages <- message{"", "", fmt.Sprintf("%s @%s acabou de entrar!", clientType, cli.nickname)}
		case cli := <-leaving:
			clientType := "Usuário"
            if cli.bot {
                clientType = "Bot"
            }

			delete(clients, cli.nickname)
			close(cli.ch)
			broadcast_messages <- message{"", "", fmt.Sprintf("%s @%s saiu", clientType, cli.nickname)}
		}
	}
}

// gorrotina para o canal de comandos
func commandsManeger() {
	for {
		select {
		case cmd := <-commands:
			if handler, ok := availableCommands[cmd.command]; ok {
				handler(cmd)
			} else {
				private_massages <- message{"", cmd.from, "Comando inválido!"}
			}
		}
	}
}

// gorrotina para os canais de mensagens
func messages() {
	for {
		select {
		case msg := <-broadcast_messages:
			for _, cli := range clients {
				if !cli.bot && cli.nickname != msg.from { // se não for bot manda a mensagem
					cli.ch <- msg.content
				}
			}
		case msg := <-private_massages:
			if cli, ok := clients[msg.to]; ok {
				var mensagem string

				if msg.from == "" {
					mensagem = msg.content
				} else {
					mensagem = fmt.Sprintf("@%s disse em privado: %s", msg.from, msg.content)
					log.Printf("Mensagem de @%s para @%s: %s", msg.from, msg.to, msg.content)
				}

				cli.ch <- mensagem
			}
		}
	}
}

// função para trocar o nick
func handleChangeNick(cmd command) {
	// verifica se tem somente um argumento
	if len(cmd.args) != 1 {
		private_massages <- message{"", cmd.from, "Uso: \\changenick <novo_apelido>"}
		return
	}

	oldNick := cmd.from
	newNick := cmd.args[0]

	// verifica se o nick está em uso
	if _, exists := clients[newNick]; exists {
		private_massages <- message{"", oldNick, fmt.Sprintf("O apelido @%s já está em uso", newNick)}
		return
	}

	cli := clients[oldNick]
	delete(clients, oldNick) // remove o usuario com nick antigo
	cli.nickname = newNick
	clients[newNick] = cli // adiciona o usuario com o nick novo
	broadcast_messages <- message{"", "", fmt.Sprintf("Usuário @%s mudou seu apelido para @%s", oldNick, newNick)}
}

// função de comando para mandar mensagem privada
func handleMsg(cmd command) {
	// verifica se ta no formato correto
	if len(cmd.args) < 2 {
		private_massages <- message{"", cmd.from, "Uso: \\msg <@destinatário> <mensagem>"}
		return
	}

	target := cmd.args[0]
	content := strings.Join(cmd.args[1:], " ")

	// verifica se o usuario começa com @
	if strings.HasPrefix(target, "@") {
		target = target[1:]
		_, exists := clients[target]

		// verifica se o usuario existe
		if exists {
			private_massages <- message{cmd.from, target, content}
		} else {
			private_massages <- message{"", cmd.from, fmt.Sprintf("Usuário @%s não encontrado", target)}
			broadcast_messages <- message{cmd.from, "", fmt.Sprintf("@%s disse: %s", cmd.from, content)}
		}
	} else {
		broadcast_messages <- message{cmd.from, "", fmt.Sprintf("@%s disse: %s", cmd.from, content)}
	}
}

// função para o comando de help
func handleHelp(cmd command) {
	helpMsg := `Comandos disponíveis:
\changenick <novo_apelido> - Muda seu apelido
\msg <@destinatário> <mensagem> - Envia uma mensagem privada
\msg <mensagem> - Envia uma mensagem para todos
\help - Mostra esta mensagem de ajuda
\exit - Sai do chat`
	private_massages <- message{"", cmd.from, helpMsg}
}

// função para escrever no console do client
func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}

// função que gerencia a conexão com o client
func handleClientConn(conn net.Conn, isBot bool) {
	defer conn.Close() // somente vai ser chamado quando a função terminar

	ch := make(chan string)   // criação do canal de mensagem
	go clientWriter(conn, ch) // atribui ao canal de mensagens o console do cliente

	var nickname string = ""
	
	input := bufio.NewScanner(conn)

	if isBot {
		nickname = "Bot"
	} else {
		// definição do nickname ao usuario entrar
		ch <- "Digite seu nickname"

		for input.Scan() {
			nickname = strings.TrimSpace(input.Text())

			if _, exists := clients[nickname]; exists {
				ch <- fmt.Sprintf("O apelido @%s já está em uso", nickname)
			} else if nickname == "" {
				ch <- "Nickname inválido"
			} else {
				break
			}
		}
	}

	cli := client{nickname: nickname, ch: ch, bot: isBot} // cria a struct de client
	entering <- &cli                                      // manda para o canal de entrar o client

	// loop principal do client
	for input.Scan() {
		message_client := input.Text() // pega o que foi enviado

		// verifica se é um comando
		if strings.HasPrefix(message_client, "\\") {
			parts := strings.Fields(message_client[1:])
			if len(parts) > 0 {
				cmd := command{from: cli.nickname, command: parts[0], args: parts[1:]}
				if cmd.command == "exit" {
					break
				} else {
					commands <- cmd
				}
			}
		} else {
			broadcast_messages <- message{cli.nickname, "", fmt.Sprintf("@%s disse: %s", cli.nickname, message_client)}
		}
	}

	leaving <- &cli // manda o cliente para o canal de saida
}

// função principal
func main() {
    fmt.Println("Iniciando servidores...")

	go messages()
	go client_inout()
	go commandsManeger()

    // Listener para clientes
    go func() {
        listenerClient, err := net.Listen("tcp", "localhost:3000")
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("Servidor iniciado na porta 3000")
        for {
            conn, err := listenerClient.Accept()
            if err != nil {
                log.Print(err)
                continue
            }
            go handleClientConn(conn, false)
        }
    }()

    // Listener para bots
    go func() {
        listenerBot, err := net.Listen("tcp", "localhost:3001")
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("Servidor iniciado na porta 3001")
        for {
            conn, err := listenerBot.Accept()
            if err != nil {
                log.Print(err)
                continue
            }
            go handleClientConn(conn, true)
        }
    }()

    //mantem o programa rodando
    select {}
}
