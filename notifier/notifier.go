package notifier

import "fmt"

// Notifier defines the interface for sending notifications.
type Notifier interface {
	Notify(address string, message string)
}

// DummyNotifier is a dummy implementation of the Notifier interface.
// It simply prints the notifications to the console.
type DummyNotifier struct{}

func NewDunnyNotifier() Notifier {
	return &DummyNotifier{}
}

func (n *DummyNotifier) Notify(address string, message string) {
	fmt.Printf("Notification for address %s: %s\n", address, message)
}
