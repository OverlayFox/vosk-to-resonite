package resonite

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity; adjust as needed for security
	},
}

type WebSocketServer struct {
	logger zerolog.Logger
	config WebSocketServerConfig

	commandChan chan Command
	listener    net.Listener

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type WebSocketServerConfig struct {
	Port int
}

func NewWebSocketServer(log zerolog.Logger, config WebSocketServerConfig) (*WebSocketServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return nil, fmt.Errorf("Failed to start listener: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketServer{
		logger: log.With().Str("component", "WebSocketServer").Logger(),
		config: config,

		commandChan: make(chan Command, 100),
		listener:    listener,

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (s *WebSocketServer) handleResoniteConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to upgrade connection")
		return
	}
	defer conn.Close()

	s.logger.Info().Msg("New WebSocket connection established")

	clientDisconnected := make(chan struct{})
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(clientDisconnected)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					s.logger.Error().Err(err).Msg("Client disconnected with error")
				}
				break
			}
			s.logger.Info().Str("message", string(message)).Msg("Received message from client")
		}
	}()

	for {
		select {
		case <-clientDisconnected:
			s.logger.Info().Msg("Client disconnected")
			return

		case <-s.ctx.Done():
			s.logger.Info().Msg("Server shutting down, closing client connection")

			shutdownMsg := websocket.FormatCloseMessage(websocket.CloseServiceRestart, "Server is shutting down")
			conn.SetWriteDeadline(time.Now().Add(time.Second * 1))
			conn.WriteMessage(websocket.CloseMessage, shutdownMsg)
			return

		case message := <-s.commandChan:
			s.logger.Debug().Interface("command", message).Msg("Sending message to client")
			if err := conn.SetWriteDeadline(time.Now().Add(time.Second * 5)); err != nil {
				s.logger.Error().Err(err).Msg("Failed to set write deadline. Disconnecting client.")
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, []byte(message.ToCommandString())); err != nil {
				s.logger.Error().Err(err).Msg("Failed to write message to client. Disconnecting client.")
				return
			}
		}
	}
}

func (s *WebSocketServer) Write(message Command) error {
	s.logger.Debug().Interface("command", message).Msg("Queueing message to send to client")

	select {
	case <-s.ctx.Done():
		return fmt.Errorf("WebSocket server is closed")
	case s.commandChan <- message:
		return nil
	default:
		return fmt.Errorf("Sending channel is full")
	}
}

func (s *WebSocketServer) Start() error {
	handler := http.NewServeMux()
	handler.HandleFunc("/ws", s.handleResoniteConnection)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := http.Serve(s.listener, handler) // will block until error occurs
		if err != nil {
			s.logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	return nil
}

func (s *WebSocketServer) Close() {
	s.logger.Info().Msg("Closing WebSocket server")

	s.cancel()
	s.listener.Close()
	s.wg.Wait()
	close(s.commandChan)

	s.logger.Info().Msg("WebSocket server closed")
}
