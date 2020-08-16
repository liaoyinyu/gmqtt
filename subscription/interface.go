package subscription

import (
	"strings"

	"github.com/DrmagicE/gmqtt/pkg/packets"
)

type Type byte

const (
	TypeSYS = iota
	TypeShared
	TypeNonShared
	TypeAll = TypeSYS | TypeShared | TypeNonShared
)

type MatchType byte

const (
	MatchName MatchType = iota
	MatchFilter
)

type Subscription interface {
	// ShareName is the share name of a shared subscription.
	// If it is a non-shared subscription, return ""
	ShareName() string
	// TopicFilter return the topic filter which does not include the share name
	TopicFilter() string
	ID() uint32 // subscription identifier
	SubOpts
}
type SubOpts interface {
	QoS() byte
	NoLocal() bool
	RetainAsPublished() bool
	RetainHandling() byte
}

type Sub struct {
	shareName   string
	topicFilter string
	id          uint32
	qos         byte
	noLocal     bool
	rap         bool
	rh          byte
}

func (s *Sub) ShareName() string {
	return s.shareName
}

func (s *Sub) TopicFilter() string {
	return s.topicFilter
}

func (s *Sub) ID() uint32 {
	return s.id
}

func (s *Sub) QoS() byte {
	return s.qos
}

func (s *Sub) NoLocal() bool {
	return s.noLocal
}

func (s *Sub) RetainAsPublished() bool {
	return s.rap
}

func (s *Sub) RetainHandling() byte {
	return s.rh
}

type subOptions func(sub *Sub)

// ID sets subscriptionIdentifier flag to the subscription
func ID(id uint32) subOptions {
	return func(sub *Sub) {
		sub.id = id
	}
}

// ShareName sets shareName of a shared subscription.
func ShareName(shareName string) subOptions {
	return func(sub *Sub) {
		sub.shareName = shareName
	}
}

func NoLocal(noLocal bool) subOptions {
	return func(sub *Sub) {
		sub.noLocal = noLocal
	}
}

func RetainAsPublished(rap bool) subOptions {
	return func(sub *Sub) {
		sub.rap = rap
	}
}

func RetainHandling(rh byte) subOptions {
	return func(sub *Sub) {
		sub.rh = rh
	}
}

// New creates a subscription
func New(topicFilter string, qos uint8, opts ...subOptions) Subscription {
	s := &Sub{
		topicFilter: topicFilter,
		qos:         qos,
	}
	for _, v := range opts {
		v(s)
	}
	return s
}

func FromTopic(topic packets.Topic, id uint32) Subscription {
	var shareName string
	var topicFilter string
	if strings.HasPrefix(topic.Name, "$share/") {
		shared := strings.SplitN(topic.Name, "/", 3)
		shareName = shared[1]
		topicFilter = shared[2]
	} else {
		topicFilter = topic.Name
	}

	s := &Sub{
		shareName:   shareName,
		topicFilter: topicFilter,
		id:          id,
		qos:         topic.Qos,
		noLocal:     topic.NoLocal,
		rap:         topic.RetainAsPublished,
		rh:          topic.RetainHandling,
	}
	return s
}

// IterateFn is the callback function used by Iterate()
// Return false means to stop the iteration.
type IterateFn func(clientID string, sub Subscription) bool

// SubscribeResult is the result of Subscribe()
type SubscribeResult = []struct {
	// Topic is the Subscribed topic
	Subscription Subscription
	// AlreadyExisted shows whether the topic is already existed.
	AlreadyExisted bool
}

// Stats is the statistics information of the store
type Stats struct {
	// SubscriptionsTotal shows how many subscription has been added to the store.
	// Duplicated subscription is not counting.
	SubscriptionsTotal uint64
	// SubscriptionsCurrent shows the current subscription number in the store.
	SubscriptionsCurrent uint64
}

// ClientSubscriptions groups the subscriptions by client id.
type ClientSubscriptions map[string][]Subscription

//
type IterationOptions struct {
	Type     Type
	ClientID string
	// TopicName filter or name
	TopicName string
	// 指定topicName的时候才有效
	MatchType MatchType
}

// Store is the interface used by gmqtt.server and external logic to handler the operations of subscriptions.
// User can get the implementation from gmqtt.Server interface.
// This interface provides the ability for extensions to interact with the subscriptions.
// Notice:
// This methods will not trigger any gmqtt hooks.
type Store interface {
	// Subscribe add subscriptions to a specific client.
	// Notice:
	// This method will succeed even if the client is not exists, the subscriptions
	// will affect the new client with the client id.
	Subscribe(clientID string, subscriptions ...Subscription) (rs SubscribeResult)
	// Unsubscribe remove subscriptions of a specific client.
	Unsubscribe(clientID string, topics ...string)
	// UnsubscribeAll remove all subscriptions of a specific client.
	UnsubscribeAll(clientID string)
	// Iterate iterate all subscriptions. The callback is called once for each subscription.
	// If callback return false, the iteration will be stopped.
	// Notice:
	// The results are not sorted in any way, no ordering of any kind is guaranteed.
	// This method will walk through all subscriptions,
	// so it is a very expensive operation. Do not call it frequently.
	Iterate(fn IterateFn, options IterationOptions)

	StatsReader
}

// GetTopicMatched returns the subscriptions that match the passed topic.
func GetTopicMatched(store Store, topicFilter string, t Type) ClientSubscriptions {
	rs := make(ClientSubscriptions)
	store.Iterate(func(clientID string, subscription Subscription) bool {
		rs[clientID] = append(rs[clientID], subscription)
		return true
	}, IterationOptions{
		Type:      t,
		TopicName: topicFilter,
		MatchType: MatchFilter,
	})
	if len(rs) == 0 {
		return nil
	}
	return rs
}

// Get returns the subscriptions that equals the passed topic filter.
func Get(store Store, topicFilter string, t Type) ClientSubscriptions {
	rs := make(ClientSubscriptions)
	store.Iterate(func(clientID string, subscription Subscription) bool {
		rs[clientID] = append(rs[clientID], subscription)
		return true
	}, IterationOptions{
		Type:      t,
		TopicName: topicFilter,
		MatchType: MatchName,
	})
	if len(rs) == 0 {
		return nil
	}
	return rs
}

// GetClientSubscriptions returns the subscriptions of a specific client.
func GetClientSubscriptions(store Store, clientID string, t Type) []Subscription {
	var rs []Subscription
	store.Iterate(func(clientID string, subscription Subscription) bool {
		rs = append(rs, subscription)
		return true
	}, IterationOptions{
		Type:     t,
		ClientID: clientID,
	})
	return rs
}

// StatsReader provides the ability to get statistics information.
type StatsReader interface {
	// GetStats return the global stats.
	GetStats() Stats
	// GetClientStats return the stats of a specific client.
	// If stats not exists, return an error.
	GetClientStats(clientID string) (Stats, error)
}
