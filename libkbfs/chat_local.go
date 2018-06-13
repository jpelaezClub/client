// Copyright 2018 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package libkbfs

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/keybase/client/go/logger"
	"github.com/keybase/client/go/protocol/chat1"
	"github.com/keybase/kbfs/kbfscrypto"
	"github.com/keybase/kbfs/tlf"
	"github.com/pkg/errors"
)

type convLocal struct {
	convID   chat1.ConversationID
	chanName string
	messages []string
	cbs      []ChatChannelNewMessageCB
	mtime    time.Time
}

type convLocalByIDMap map[string]*convLocal

type convLocalByTypeMap map[tlf.Type]convLocalByIDMap

type convLocalByNameMap map[tlf.CanonicalName]convLocalByTypeMap

type newConvCB func(context.Context, *TlfHandle, chat1.ConversationID, string)

type chatLocalSharedData struct {
	lock          sync.Mutex
	newChannelCBs []newConvCB
	convs         convLocalByNameMap
	convsByID     convLocalByIDMap
}

// chatLocal is a local implementation for chat.
type chatLocal struct {
	config   Config
	log      logger.Logger
	deferLog logger.Logger
	data     *chatLocalSharedData
}

func newChatLocalWithData(config Config, data *chatLocalSharedData) *chatLocal {
	log := config.MakeLogger("")
	deferLog := log.CloneWithAddedDepth(1)
	return &chatLocal{
		log:      log,
		deferLog: deferLog,
		config:   config,
		data:     data,
	}
}

// newChatLocal constructs a new local chat implementation.
func newChatLocal(config Config) *chatLocal {
	return newChatLocalWithData(config, &chatLocalSharedData{
		convs:         make(convLocalByNameMap),
		convsByID:     make(convLocalByIDMap),
		newChannelCBs: []newConvCB{config.KBFSOps().NewNotificationChannel},
	})
}

var _ Chat = (*chatLocal)(nil)

// GetConversationID implements the Chat interface.
func (c *chatLocal) GetConversationID(
	ctx context.Context, tlfName tlf.CanonicalName, tlfType tlf.Type,
	channelName string, chatType chat1.TopicType) (
	chat1.ConversationID, error) {
	if chatType != chat1.TopicType_KBFSFILEEDIT {
		panic(fmt.Sprintf("Bad topic type: %d", chatType))
	}

	c.data.lock.Lock()
	defer c.data.lock.Unlock()
	byID, ok := c.data.convs[tlfName][tlfType]
	if !ok {
		if _, ok := c.data.convs[tlfName]; !ok {
			c.data.convs[tlfName] = make(convLocalByTypeMap)
		}
		if _, ok := c.data.convs[tlfName][tlfType]; !ok {
			byID = make(convLocalByIDMap)
			c.data.convs[tlfName][tlfType] = byID
		}
	}
	for _, conv := range byID {
		if conv.chanName == channelName {
			return conv.convID, nil
		}
	}
	// Make a new conversation.
	var idBytes [8]byte
	err := kbfscrypto.RandRead(idBytes[:])
	if err != nil {
		return nil, err
	}
	id := chat1.ConversationID(idBytes[:])
	conv := &convLocal{
		convID:   id,
		chanName: channelName,
	}
	c.data.convs[tlfName][tlfType][id.String()] = conv
	c.data.convsByID[id.String()] = conv
	return id, nil
}

// SendTextMessage implements the Chat interface.
func (c *chatLocal) SendTextMessage(
	ctx context.Context, tlfName tlf.CanonicalName, tlfType tlf.Type,
	convID chat1.ConversationID, body string) error {
	c.data.lock.Lock()
	defer c.data.lock.Unlock()
	conv, ok := c.data.convs[tlfName][tlfType][convID.String()]
	if !ok {
		return errors.Errorf("Conversation %s doesn't exist", convID.String())
	}
	conv.messages = append(conv.messages, body)
	conv.mtime = c.config.Clock().Now()
	for _, cb := range conv.cbs {
		cb(convID, body)
	}
	return nil
}

type chatHandleAndTime struct {
	h     *TlfHandle
	mtime time.Time
}

// GetGroupedInbox implements the Chat interface.
func (c *chatLocal) GetGroupedInbox(
	ctx context.Context, chatType chat1.TopicType, maxChats int) (
	results []*TlfHandle, err error) {
	if chatType != chat1.TopicType_KBFSFILEEDIT {
		panic(fmt.Sprintf("Bad topic type: %d", chatType))
	}

	session, err := c.config.KBPKI().GetCurrentSession(ctx)
	if err != nil {
		return nil, err
	}

	var handlesAndTimes []chatHandleAndTime

	c.data.lock.Lock()
	defer c.data.lock.Unlock()
	for name, byType := range c.data.convs {
		for t, byID := range byType {
			if t == tlf.Public && string(name) != string(session.Name) {
				// Skip public TLFs that aren't our own.
				continue
			}

			h, err := GetHandleFromFolderNameAndType(
				ctx, c.config.KBPKI(), c.config.MDOps(), string(name), t)
			if err != nil {
				return nil, err
			}

			// Only include if the current user can read the folder.
			isReader, err := isReaderFromHandle(
				ctx, h, c.config.KBPKI(), session.UID)
			if err != nil {
				return nil, err
			}
			if !isReader {
				continue
			}

			hAndT := chatHandleAndTime{h: h}
			for _, conv := range byID {
				if conv.mtime.After(hAndT.mtime) {
					hAndT.mtime = conv.mtime
				}
			}
			handlesAndTimes = append(handlesAndTimes, hAndT)
		}
	}

	sort.Slice(handlesAndTimes, func(i, j int) bool {
		// Reverse sort so newest conversation is at index 0.
		return handlesAndTimes[i].mtime.After(handlesAndTimes[j].mtime)
	})
	for i := 0; i < maxChats && i < len(handlesAndTimes); i++ {
		results = append(results, handlesAndTimes[i].h)
	}
	return results, nil
}

// GetChannels implements the Chat interface.
func (c *chatLocal) GetChannels(
	ctx context.Context, tlfName tlf.CanonicalName, tlfType tlf.Type,
	chatType chat1.TopicType) (
	convIDs []chat1.ConversationID, channelNames []string, err error) {
	if chatType != chat1.TopicType_KBFSFILEEDIT {
		panic(fmt.Sprintf("Bad topic type: %d", chatType))
	}

	c.data.lock.Lock()
	defer c.data.lock.Unlock()
	byID := c.data.convs[tlfName][tlfType]
	for _, conv := range byID {
		convIDs = append(convIDs, conv.convID)
		channelNames = append(channelNames, conv.chanName)
	}
	return convIDs, channelNames, nil
}

// ReadChannel implements the Chat interface.
func (c *chatLocal) ReadChannel(
	ctx context.Context, convID chat1.ConversationID, startPage []byte) (
	messages []string, nextPage []byte, err error) {
	c.data.lock.Lock()
	defer c.data.lock.Unlock()
	conv, ok := c.data.convsByID[convID.String()]
	if !ok {
		return nil, nil, errors.Errorf(
			"Conversation %s doesn't exist", convID.String())
	}
	// For now, no paging, just return the complete list.
	return conv.messages, nil, nil
}

// RegisterForMessages implements the Chat interface.
func (c *chatLocal) RegisterForMessages(
	convID chat1.ConversationID, cb ChatChannelNewMessageCB) {
	c.data.lock.Lock()
	defer c.data.lock.Unlock()
	conv, ok := c.data.convsByID[convID.String()]
	if !ok {
		panic(fmt.Sprintf("Conversation %s doesn't exist", convID.String()))
	}
	conv.cbs = append(conv.cbs, cb)
}

func (c *chatLocal) copy(config Config) *chatLocal {
	copy := newChatLocalWithData(config, c.data)
	c.data.lock.Lock()
	defer c.data.lock.Unlock()
	c.data.newChannelCBs = append(
		c.data.newChannelCBs, config.KBFSOps().NewNotificationChannel)
	return copy
}
