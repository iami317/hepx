package msg

import (
	"io"
	"reflect"
)

func AsyncHandler(f func(Message)) func(Message) {
	return func(m Message) {
		go f(m)
	}
}

// Dispatcher is used to send messages to net.Conn or register handlers for messages read from net.Conn.
type Dispatcher struct {
	rw io.ReadWriter

	sendCh         chan Message
	doneCh         chan struct{}
	msgHandlers    map[reflect.Type]func(Message)
	defaultHandler func(Message)
}

func NewDispatcher(rw io.ReadWriter) *Dispatcher {
	return &Dispatcher{
		rw:          rw,
		sendCh:      make(chan Message, 100),
		doneCh:      make(chan struct{}),
		msgHandlers: make(map[reflect.Type]func(Message)),
	}
}

// Run will block until io.EOF or some error occurs.
func (d *Dispatcher) Run() {
	go d.sendLoop()
	go d.readLoop()
}

func (d *Dispatcher) sendLoop() {
	for {
		select {
		case <-d.doneCh:
			return
		case m := <-d.sendCh:
			_ = WriteMsg(d.rw, m)
		}
	}
}

func (d *Dispatcher) readLoop() {
	for {
		m, err := ReadMsg(d.rw)
		if err != nil {
			close(d.doneCh)
			return
		}

		if handler, ok := d.msgHandlers[reflect.TypeOf(m)]; ok {
			handler(m)
		} else if d.defaultHandler != nil {
			d.defaultHandler(m)
		}
	}
}

func (d *Dispatcher) Send(m Message) error {
	select {
	case <-d.doneCh:
		return io.EOF
	case d.sendCh <- m:
		return nil
	}
}

func (d *Dispatcher) SendChannel() chan Message {
	return d.sendCh
}

func (d *Dispatcher) RegisterHandler(msg Message, handler func(Message)) {
	d.msgHandlers[reflect.TypeOf(msg)] = handler
}

func (d *Dispatcher) RegisterDefaultHandler(handler func(Message)) {
	d.defaultHandler = handler
}

func (d *Dispatcher) Done() chan struct{} {
	return d.doneCh
}
