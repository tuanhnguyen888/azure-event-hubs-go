package eventhub_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tuanhnguyen888/azure-event-hubs-go"
)

func TestNewEventBatch(t *testing.T) {
	eb := eventhub.NewEventBatch("eventId", nil)
	assert.Equal(t, eventhub.DefaultMaxMessageSizeInBytes, eb.MaxSize)
}

func TestEventBatch_AddOneMessage(t *testing.T) {
	eb := eventhub.NewEventBatch("eventId", nil)
	event := eventhub.NewEventFromString("Foo")
	ok, err := eb.Add(event)
	assert.True(t, ok)
	assert.NoError(t, err)
}

func TestEventBatch_AddManyMessages(t *testing.T) {
	eb := eventhub.NewEventBatch("eventId", nil)
	wrapperSize := eb.Size()
	event := eventhub.NewEventFromString("Foo")
	ok, err := eb.Add(event)
	assert.True(t, ok)
	assert.NoError(t, err)
	msgSize := eb.Size() - wrapperSize

	limit := ((int(eb.MaxSize) - 100) / msgSize) - 1
	for i := 0; i < limit; i++ {
		ok, err := eb.Add(event)
		assert.True(t, ok)
		assert.NoError(t, err)
	}

	ok, err = eb.Add(event)
	assert.False(t, ok)
	assert.NoError(t, err)
}

func TestEventBatch_Clear(t *testing.T) {
	eb := eventhub.NewEventBatch("eventId", nil)
	ok, err := eb.Add(eventhub.NewEventFromString("Foo"))
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, 174, eb.Size())

	eb.Clear()
	assert.Equal(t, 100, eb.Size())
}
