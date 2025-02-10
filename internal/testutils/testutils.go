package testutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func GetFreePort() (string, error) {
	listener, err := net.Listen("tcp", ":0") // Listen on any available port by passing port 0
	if err != nil {
		return "", err
	}
	defer listener.Close() // Make sure the listener is closed when we're done
	address := listener.Addr().String()
	port := address[strings.LastIndex(address, ":")+1:]
	return port, nil
}

func WaitForPort(port string, timeout time.Duration) error {
	start := time.Now()
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", port))
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("port %s did not become ready within %v", port, timeout)
		}
		time.Sleep(100 * time.Millisecond) // Polling interval
	}
}

func WaitForServerLog(reader io.Reader, logMessage string, timeout time.Duration) error {
	scanner := bufio.NewScanner(reader)
	start := time.Now()
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, logMessage) {
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("server did not log '%s' within %v", logMessage, timeout)
		}
	}
	return scanner.Err()
}

// WaitForConsumerGroup waits for the Kafka consumer group to become active (Stable state with members)
func WaitForConsumerGroup(ctx context.Context, broker, groupID string, timeout time.Duration) error {
	if broker == "" || groupID == "" {
		return fmt.Errorf("broker and groupID must not be empty")
	}

	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	ctx, cancel := context.WithTimeout(ctx, timeout) // Set up a context with the specified timeout
	defer cancel()

	pollInterval := 2 * time.Second // Polling interval for checking the consumer group status
	for {                           // Loop until the context times out or the group becomes active
		select {
		case <-ctx.Done(): // Timeout reached
			return fmt.Errorf("timed out waiting for consumer group %s to become active", groupID)
		default:
			groupMetadata, err := adminClient.DescribeConsumerGroups(ctx, []string{groupID})
			if err != nil {
				continue // continue on failed to describe consumer group
			}

			// Check if the group exists and is active
			for _, group := range groupMetadata.ConsumerGroupDescriptions {
				if group.Error.Code() != kafka.ErrNoError { // Handle group-specific errors (e.g., group doesn't exist)
					continue // continue on failed to describe consumer group metadata
				} else {
					return nil // consumer group is active
				}
				// if group.State == kafka.ConsumerGroupStateStable && len(group.Members) > 0 {
				// 	return nil // consumer group is active
				// }
			}
			time.Sleep(pollInterval)
		}
	}
}
