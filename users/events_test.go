package users

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/delicb/toy-cqrs/cqrs"
)

func TestSerializationUserCreated(t *testing.T) {
	origEv := &cqrs.Event{
		EventID:       UserCreatedID,
		AggregateID:   "someAgg",
		CreatedAt:     time.Now().UTC(),
		CorrelationID: "corr",
		Data:          &UserCreated{
			ID:        "someAgg",
			Email:     "bojan@delic.in.rs",
			Password:  "somePass",
			IsEnabled: false,
		},
	}
	raw, err := json.Marshal(origEv)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(raw))

	ev, err := EventSerializer.Unmarshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", ev)
	t.Logf("%+v\n", ev.Data)
}

func TestSerializationEmail(t *testing.T) {
	origEv := &cqrs.Event{
		EventID:       EmailChangedID,
		AggregateID:   "someAgg",
		CreatedAt:     time.Now().UTC(),
		CorrelationID: "corr",
		Data:          &UserEmailChanged{
			NewEmail: "delicb@gmail.com",
			OldEmail: "bojan@delic.in.rs",
		},
	}
	raw, err := json.Marshal(origEv)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(raw))

	ev, err := EventSerializer.Unmarshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", ev)
	t.Logf("%+v\n", ev.Data)
}

func TestSerializationPassword(t *testing.T) {
	origEv := &cqrs.Event{
		EventID:       PasswordChangedID,
		AggregateID:   "someAgg",
		CreatedAt:     time.Now().UTC(),
		CorrelationID: "corr",
		Data:          &UserPasswordChanged{
			NewPassword: "newPass",
			OldPassword: "oldPass",
		},
	}
	raw, err := json.Marshal(origEv)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(raw))

	ev, err := EventSerializer.Unmarshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", ev)
	t.Logf("%+v\n", ev.Data)
}
