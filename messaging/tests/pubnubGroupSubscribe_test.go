// Package tests has the unit tests of package messaging.
// pubnubGroupSubscribe_test.go contains the tests related to the Group
// Subscribe requests on pubnub Api
package tests

import (
	"encoding/json"
	"fmt"

	"github.com/pubnub/go/messaging"
	"github.com/pubnub/go/messaging/tests/utils"
	"github.com/stretchr/testify/assert"
	// "os"
	"strings"
	"testing"
	"time"
)

// TestGroupSubscribeStart prints a message on the screen to mark the beginning of
// subscribe tests.
// PrintTestMessage is defined in the common.go file.
func TestGroupSubscribeStart(t *testing.T) {
	PrintTestMessage("==========Group Subscribe tests start==========")
}

func TestGroupSubscriptionConnectedAndUnsubscribedSingle(t *testing.T) {
	assert := assert.New(t)

	stop, sleep := NewVCRBothWithSleep(
		"fixtures/groups/conAndUnsSingle", []string{"uuid"}, 2)
	defer stop()

	group := "Group_GroupSubscriptionConAndUnsSingle"
	pubnubInstance := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	createChannelGroups(pubnubInstance, []string{group})
	defer removeChannelGroups(pubnubInstance, []string{group})

	sleep(2)

	subscribeSuccessChannel := make(chan []byte)
	subscribeErrorChannel := make(chan []byte)
	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)

	go pubnubInstance.ChannelGroupSubscribe(group,
		subscribeSuccessChannel, subscribeErrorChannel)
	select {
	case msg := <-subscribeSuccessChannel:
		val := string(msg)
		assert.Equal(val, fmt.Sprintf(
			"[1, \"Subscription to channel group '%s' connected\", \"%s\"]",
			group, group))
	case err := <-subscribeErrorChannel:
		assert.Fail(string(err))
	}

	go pubnubInstance.ChannelGroupUnsubscribe(group, successChannel, errorChannel)
	select {
	case msg := <-successChannel:
		val := string(msg)
		assert.Equal(val, fmt.Sprintf(
			"[1, \"Subscription to channel group '%s' unsubscribed\", \"%s\"]",
			group, group))
	case err := <-errorChannel:
		assert.Fail(string(err))
	}

	select {
	case ev := <-successChannel:
		var event messaging.PresenceResonse

		err := json.Unmarshal(ev, &event)
		if err != nil {
			assert.Fail(err.Error())
		}

		assert.Equal("leave", event.Action)
		assert.Equal(200, event.Status)
	case err := <-errorChannel:
		assert.Fail(string(err))
	}

	pubnubInstance.CloseExistingConnection()
}

func TestGroupSubscriptionConnectedAndUnsubscribedMultiple(t *testing.T) {
	assert := assert.New(t)
	pubnub := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")
	groupsString, _ := GenerateTwoRandomChannelStrings(3)
	groups := strings.Split(groupsString, ",")

	createChannelGroups(pubnub, groups)
	defer removeChannelGroups(pubnub, groups)

	time.Sleep(1 * time.Second)

	subscribeSuccessChannel := make(chan []byte)
	subscribeErrorChannel := make(chan []byte)
	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	await := make(chan bool)

	go pubnub.ChannelGroupSubscribe(groupsString,
		subscribeSuccessChannel, subscribeErrorChannel)

	go func() {
		var messages []string

		for {
			select {
			case message := <-subscribeSuccessChannel:
				var msg []interface{}

				err := json.Unmarshal(message, &msg)
				if err != nil {
					assert.Fail(err.Error())
				}

				assert.Contains(msg[1].(string), "Subscription to channel group")
				assert.Contains(msg[1].(string), "connected")
				assert.Len(msg, 3)

				messages = append(messages, string(msg[2].(string)))
			case err := <-subscribeErrorChannel:
				assert.Fail("Subscribe error", string(err))
			case <-timeouts(10):
				break
			}

			if len(messages) == 3 {
				break
			}
		}

		assert.True(utils.AssertStringSliceElementsEqual(groups, messages),
			fmt.Sprintf("Expected groups: %s. Actual groups: %s\n", groups, messages))

		await <- true
	}()

	select {
	case <-await:
	case <-timeouts(20):
		assert.Fail("Receive connected messages timeout")
	}

	go pubnub.ChannelGroupUnsubscribe(groupsString, successChannel, errorChannel)
	go func() {
		var messages []string

		for {
			select {
			case message := <-successChannel:
				var msg []interface{}

				err := json.Unmarshal(message, &msg)
				if err != nil {
					assert.Fail(err.Error())
				}

				assert.Contains(msg[1].(string), "Subscription to channel group")
				assert.Contains(msg[1].(string), "unsubscribed")
				assert.Len(msg, 3)

				messages = append(messages, string(msg[2].(string)))
			case err := <-errorChannel:
				assert.Fail("Subscribe error", string(err))
			case <-timeouts(10):
				break
			}

			if len(messages) == 3 {
				break
			}
		}

		assert.True(utils.AssertStringSliceElementsEqual(groups, messages),
			fmt.Sprintf("Expected groups: %s. Actual groups: %s\n", groups, messages))

		await <- true
	}()

	select {
	case <-await:
	case <-timeouts(20):
		assert.Fail("Receive unsubscribed messages timeout")
	}

	pubnub.CloseExistingConnection()
}

func TestGroupSubscriptionReceiveSingleMessage(t *testing.T) {
	assert := assert.New(t)

	stop, sleep := NewVCRBothWithSleep(
		"fixtures/groups/receiveSingleMessage", []string{"uuid"}, 3)
	defer stop()

	group := "Group_GroupReceiveSingleMessage"
	channel := "Channel_GroupReceiveSingleMessage"
	pubnub := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")

	populateChannelGroup(pubnub, group, channel)
	defer removeChannelGroups(pubnub, []string{group})

	sleep(2)

	subscribeSuccessChannel := make(chan []byte)
	subscribeErrorChannel := make(chan []byte)
	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)

	msgReceived := make(chan bool)

	go pubnub.ChannelGroupSubscribe(group,
		subscribeSuccessChannel, subscribeErrorChannel)
	ExpectConnectedEvent(t, "", group, subscribeSuccessChannel,
		subscribeErrorChannel)

	go func() {
		select {
		case message := <-subscribeSuccessChannel:
			var msg []interface{}

			err := json.Unmarshal(message, &msg)
			if err != nil {
				assert.Fail(err.Error())
			}

			assert.Len(msg, 4)
			assert.Equal(msg[2], channel)
			assert.Equal(msg[3], group)
			msgReceived <- true
		case err := <-subscribeErrorChannel:
			assert.Fail(string(err))
		case <-timeouts(3):
			assert.Fail("Subscription timeout")
		}
	}()

	go pubnub.Publish(channel, "hey", successChannel, errorChannel)
	select {
	case <-successChannel:
	case err := <-errorChannel:
		assert.Fail("Publish error", string(err))
	case <-messaging.Timeout():
		assert.Fail("Publish timeout")
	}

	<-msgReceived

	go pubnub.ChannelGroupUnsubscribe(group, unsubscribeSuccessChannel,
		unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, "", group, unsubscribeSuccessChannel,
		unsubscribeErrorChannel)

	pubnub.CloseExistingConnection()
}

func TestGroupSubscriptionPresence(t *testing.T) {
	presenceTimeout := 15
	assert := assert.New(t)
	pubnub := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")
	group := RandomChannel()
	groupPresence := fmt.Sprintf("%s%s", group, presenceSuffix)

	createChannelGroups(pubnub, []string{group})
	defer removeChannelGroups(pubnub, []string{group})

	time.Sleep(1 * time.Second)

	presenceSuccessChannel := make(chan []byte)
	presenceErrorChannel := make(chan []byte)
	subscribeSuccessChannel := make(chan []byte)
	subscribeErrorChannel := make(chan []byte)
	unsubscribeSuccessChannel := make(chan []byte)
	unsubscribeErrorChannel := make(chan []byte)

	await := make(chan bool)

	go pubnub.ChannelGroupSubscribe(groupPresence,
		presenceSuccessChannel, presenceErrorChannel)
	ExpectConnectedEvent(t, "", group, presenceSuccessChannel,
		presenceErrorChannel)

	go func() {
		for {
			select {
			case message := <-presenceSuccessChannel:
				var msg []interface{}

				msgString := string(message)

				err := json.Unmarshal(message, &msg)
				if err != nil {
					assert.Fail(err.Error())
				}

				if strings.Contains(msgString, "timeout") ||
					strings.Contains(msgString, "leave") {
					continue
				}

				assert.Equal("adsf", msg[2].(string))
				assert.Equal(group, msg[3].(string))

				assert.Contains(msgString, "join")
				assert.Contains(msgString, pubnub.GetUUID())
				await <- true
				return
			case err := <-presenceErrorChannel:
				assert.Fail(string(err))
				await <- false
				return
			case <-timeouts(presenceTimeout):
				assert.Fail("Presence timeout")
				await <- false
				return
			}
		}
	}()

	go pubnub.ChannelGroupSubscribe(group,
		subscribeSuccessChannel, subscribeErrorChannel)
	ExpectConnectedEvent(t, "", group, subscribeSuccessChannel,
		subscribeErrorChannel)

	<-await

	time.Sleep(3 * time.Second)

	go func() {
		for {
			select {
			case message := <-presenceSuccessChannel:
				var msg []interface{}

				msgString := string(message)

				err := json.Unmarshal(message, &msg)
				if err != nil {
					assert.Fail(err.Error())
				}

				if strings.Contains(msgString, "timeout") ||
					strings.Contains(msgString, "join") {
					continue
				}

				assert.Equal("adsf", msg[2].(string))
				assert.Equal(group, msg[3].(string))

				assert.Contains(msgString, "leave")
				assert.Contains(msgString, pubnub.GetUUID())
				await <- true
				return
			case err := <-presenceErrorChannel:
				assert.Fail(string(err))
				await <- false
				return
			case <-timeouts(presenceTimeout):
				assert.Fail("Presence timeout")
				await <- false
				return
			}
		}
	}()

	go pubnub.ChannelGroupUnsubscribe(group, unsubscribeSuccessChannel,
		unsubscribeErrorChannel)
	ExpectUnsubscribedEvent(t, "", group, unsubscribeSuccessChannel,
		unsubscribeErrorChannel)

	<-await

	pubnub.CloseExistingConnection()
}

func TestGroupSubscriptionAlreadySubscribed(t *testing.T) {
	//messaging.SetLogOutput(os.Stderr)
	assert := assert.New(t)
	pubnub := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")
	group := RandomChannel()

	createChannelGroups(pubnub, []string{group})
	defer removeChannelGroups(pubnub, []string{group})

	time.Sleep(1 * time.Second)

	subscribeSuccessChannel := make(chan []byte)
	subscribeErrorChannel := make(chan []byte)
	subscribeSuccessChannel2 := make(chan []byte)
	subscribeErrorChannel2 := make(chan []byte)
	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)

	go pubnub.ChannelGroupSubscribe(group,
		subscribeSuccessChannel, subscribeErrorChannel)
	ExpectConnectedEvent(t, "", group, subscribeSuccessChannel, subscribeErrorChannel)

	go pubnub.ChannelGroupSubscribe(group,
		subscribeSuccessChannel2, subscribeErrorChannel2)
	select {
	case <-subscribeSuccessChannel2:
		assert.Fail("Received success message while expecting error")
	case err := <-subscribeErrorChannel2:
		assert.Contains(string(err), "Subscription to channel group")
		assert.Contains(string(err), "already subscribed")
	}

	go pubnub.ChannelGroupUnsubscribe(group, successChannel, errorChannel)
	ExpectUnsubscribedEvent(t, "", group, successChannel, errorChannel)

	pubnub.CloseExistingConnection()
}

func TestGroupSubscriptionNotSubscribed(t *testing.T) {
	assert := assert.New(t)
	pubnub := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")
	group := RandomChannel()

	createChannelGroups(pubnub, []string{group})
	defer removeChannelGroups(pubnub, []string{group})

	time.Sleep(1 * time.Second)

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)

	go pubnub.ChannelGroupUnsubscribe(group, successChannel, errorChannel)
	select {
	case response := <-successChannel:
		assert.Fail("Received success message while expecting error", string(response))
	case err := <-errorChannel:
		assert.Contains(string(err), "Subscription to channel group")
		assert.Contains(string(err), "not subscribed")
	}

	pubnub.CloseExistingConnection()
}

func TestGroupSubscriptionToNotExistingChannelGroup(t *testing.T) {
	assert := assert.New(t)
	pubnub := messaging.NewPubnub(PubKey, SubKey, "", "", false, "")
	group := RandomChannel()

	successChannel := make(chan []byte)
	errorChannel := make(chan []byte)

	removeChannelGroups(pubnub, []string{group})

	time.Sleep(1 * time.Second)

	go pubnub.ChannelGroupSubscribe(group, successChannel, errorChannel)
	select {
	case response := <-successChannel:
		assert.Fail("Received success message while expecting error", string(response))
	case err := <-errorChannel:
		assert.Contains(string(err), "Channel group or groups result in empty subscription set")
		assert.Contains(string(err), group)
	}

	pubnub.CloseExistingConnection()
}

func createChannelGroups(pubnub *messaging.Pubnub, groups []string) {
	successChannel := make(chan []byte, 1)
	errorChannel := make(chan []byte, 1)

	for _, group := range groups {
		// fmt.Println("Creating group", group)

		pubnub.ChannelGroupAddChannel(group, "adsf", successChannel, errorChannel)

		select {
		case <-successChannel:
			// fmt.Println("Group created")
		case <-errorChannel:
			fmt.Println("Channel group creation error")
		case <-timeout():
			fmt.Println("Channel group creation timeout")
		}
	}
}

func populateChannelGroup(pubnub *messaging.Pubnub, group, channels string) {

	successChannel := make(chan []byte, 1)
	errorChannel := make(chan []byte, 1)

	pubnub.ChannelGroupAddChannel(group, channels, successChannel, errorChannel)

	select {
	case <-successChannel:
		// fmt.Println("Group created")
	case <-errorChannel:
		fmt.Println("Channel group creation error")
	case <-timeout():
		fmt.Println("Channel group creation timeout")
	}
}

func removeChannelGroups(pubnub *messaging.Pubnub, groups []string) {
	successChannel := make(chan []byte, 1)
	errorChannel := make(chan []byte, 1)

	for _, group := range groups {
		// fmt.Println("Removing group", group)

		pubnub.ChannelGroupRemoveGroup(group, successChannel, errorChannel)

		select {
		case <-successChannel:
			// fmt.Println("Group removed")
		case <-errorChannel:
			fmt.Println("Channel group removal error")
		case <-timeout():
			fmt.Println("Channel group removal timeout")
		}
	}
}

// TestGroupSubscribeEnd prints a message on the screen to mark the end of
// subscribe tests.
// PrintTestMessage is defined in the common.go file.
func TestGroupSubscribeEnd(t *testing.T) {
	PrintTestMessage("==========Group Subscribe tests end==========")
}
