package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/seaptc/seaptc/conference"
	"github.com/seaptc/seaptc/log"
)

type Store struct {
	client *datastore.Client

	mu         sync.RWMutex
	versions   map[string]int64
	lastSync   time.Time
	maxVersion int64
	conf       *conference.Conference
}

func New(ctx context.Context, projectID string, useEmulator bool) (*Store, error) {
	const emulatorKey = "DATASTORE_EMULATOR_HOST"
	if useEmulator {
		if os.Getenv(emulatorKey) == "" {
			return nil, fmt.Errorf("Datatstore emulator host not set.\n"+
				"To start the emulator run: gcloud beta emulators datastore start\n"+
				"and export %s=host:port", emulatorKey)
		}
	} else {
		os.Unsetenv(emulatorKey)
	}

	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Store{
		client:   client,
		conf:     conference.New(),
		versions: map[string]int64{},
	}, nil
}

var conferenceEntityGroupKey = datastore.IDKey("conference", 1, nil)

func blobKey(name string) *datastore.Key {
	return &datastore.Key{Kind: "blob", Name: name, Parent: conferenceEntityGroupKey}
}

func evaluationKey(participantID string) *datastore.Key {
	return &datastore.Key{Kind: "eval", Name: participantID, Parent: conferenceEntityGroupKey}
}

type blobEntity struct {
	Version int64
	Data    []byte `datastore:",noindex"`
}

type metaEntity struct {
	Version int64
}

var (
	metaKey              = &datastore.Key{Kind: "meta", ID: 1, Parent: conferenceEntityGroupKey}
	configurationKey     = blobKey("configuration")
	classesKey           = blobKey("classes")
	participantsKey      = blobKey("participants")
	instructorClassesKey = blobKey("instructorClasses")
	loginCodesKey        = blobKey("loginCodes")
	printSignaturesKey   = blobKey("printSignatures")
)

var blobUpdaters = map[string]func(*conference.Conference, []byte) (*conference.Conference, error){
	configurationKey.Name:     updateConfiguration,
	classesKey.Name:           updateClasses,
	participantsKey.Name:      updateParticipants,
	instructorClassesKey.Name: updateInstructorClasses,
}

const maxAge = time.Minute * 10

func (s *Store) GetConference(ctx context.Context, noCache bool) (*conference.Conference, bool, error) {
	s.mu.RLock()
	conf := s.conf
	lastSync := s.lastSync
	maxVersion := s.maxVersion
	s.mu.RUnlock()

	if !noCache && time.Since(lastSync) < maxAge {
		return conf, true, nil
	}

	var blobs []blobEntity
	keys, err := s.client.GetAll(ctx,
		datastore.NewQuery("blob").Ancestor(conferenceEntityGroupKey).Filter("Version >", maxVersion),
		&blobs)
	if err != nil {
		return nil, true, fmt.Errorf("error querying for blob updates: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, b := range blobs {
		name := keys[i].Name
		if b.Version <= s.versions[name] {
			continue
		}
		fn := blobUpdaters[name]
		if fn == nil {
			return nil, true, fmt.Errorf("store: unknown blob name %q", name)
		}

		log.Logf(ctx, log.Info, "Loading blob %s", name)

		conf, err = fn(conf, b.Data)
		if err != nil {
			return nil, true, err
		}
		s.versions[name] = b.Version
		if b.Version > s.maxVersion {
			s.maxVersion = b.Version
		}
		s.conf = conf
	}

	return s.conf, false, nil
}

func (s *Store) putBlob(ctx context.Context, key *datastore.Key, data []byte) error {
	_, err := s.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var m metaEntity
		err := noEntityOK(tx.Get(metaKey, &m))
		if err != nil {
			return err
		}
		m.Version += 1
		_, err = tx.Put(key, &blobEntity{Version: m.Version, Data: data})
		if err != nil {
			return err
		}
		_, err = tx.Put(metaKey, &m)
		return err
	})
	return err
}

func updateConfiguration(conf *conference.Conference, data []byte) (*conference.Conference, error) {
	var config conference.Configuration
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return conf.UpdateConfiguration(&config), nil
}

func (s *Store) PutConfiguration(ctx context.Context, config *conference.Configuration) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return s.putBlob(ctx, configurationKey, data)
}

func updateClasses(conf *conference.Conference, data []byte) (*conference.Conference, error) {
	var classes []*conference.Class
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&classes)
	if err != nil {
		return nil, fmt.Errorf("store.classes: %w", err)
	}
	return conf.UpdateClasses(classes), nil
}

func (s *Store) PutClasses(ctx context.Context, classes []*conference.Class) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(classes)
	if err != nil {
		return err
	}
	return s.putBlob(ctx, classesKey, buf.Bytes())
}

func updateParticipants(conf *conference.Conference, data []byte) (*conference.Conference, error) {
	var participants []*conference.Participant
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&participants)
	if err != nil {
		return nil, fmt.Errorf("store.participants: %w", err)
	}
	return conf.UpdateParticipants(participants), nil
}

func (s *Store) PutParticipants(ctx context.Context, participants []*conference.Participant) error {
	_, err := s.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var m metaEntity
		err := noEntityOK(tx.Get(metaKey, &m))
		if err != nil {
			return err
		}
		var loginCodesBlob blobEntity
		err = noEntityOK(tx.Get(loginCodesKey, &loginCodesBlob))
		if err != nil {
			return err
		}

		var loginCodes map[string]string
		if len(loginCodesBlob.Data) == 0 {
			loginCodes = make(map[string]string)
		} else {
			err := gob.NewDecoder(bytes.NewReader(loginCodesBlob.Data)).Decode(&loginCodes)
			if err != nil {
				return err
			}
		}

		// To ensure that login codes do not change when a participant is
		// deleted and added again, the login codes are stored in separate
		// blob. Assigned codes are never removed from the blob.

		err = assignLoginCodes(loginCodes, participants)
		if err != nil {
			return err
		}

		m.Version += 1
		_, err = tx.Put(metaKey, &m)

		var loginCodesBuf bytes.Buffer
		err = gob.NewEncoder(&loginCodesBuf).Encode(loginCodes)
		if err != nil {
			return err
		}

		_, err = tx.Put(loginCodesKey, &blobEntity{Data: loginCodesBuf.Bytes()})
		if err != nil {
			return err
		}

		var participantsBuf bytes.Buffer
		err = gob.NewEncoder(&participantsBuf).Encode(participants)
		if err != nil {
			return err
		}

		_, err = tx.Put(participantsKey, &blobEntity{Version: m.Version, Data: participantsBuf.Bytes()})
		return err
	})
	return err
}

func updateInstructorClasses(conf *conference.Conference, data []byte) (*conference.Conference, error) {
	var instructorClasses map[string][]int
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&instructorClasses)
	if err != nil {
		return nil, fmt.Errorf("store.instructorClasses: %w", err)
	}
	return conf.UpdateInstructorClasses(instructorClasses), nil
}

func (s *Store) ModifyInstructorClasses(ctx context.Context, participantID string, modifications map[int]int) error {
	_, err := s.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var m metaEntity
		err := noEntityOK(tx.Get(metaKey, &m))
		if err != nil {
			return err
		}
		var blob blobEntity
		err = noEntityOK(tx.Get(instructorClassesKey, &blob))
		if err != nil {
			return err
		}

		m.Version += 1

		var instructorClasses map[string][]int
		if len(blob.Data) == 0 {
			instructorClasses = make(map[string][]int)
		} else {
			err := gob.NewDecoder(bytes.NewReader(blob.Data)).Decode(&instructorClasses)
			if err != nil {
				return err
			}
		}

		classNumbers := instructorClasses[participantID]
		if classNumbers == nil {
			classNumbers = make([]int, conference.NumSession)
		}

		for i, n := range modifications {
			classNumbers[i] = n
		}

		allZero := true
		for _, n := range classNumbers {
			if n != 0 {
				allZero = false
				break
			}
		}

		if allZero {
			delete(instructorClasses, participantID)
		} else {
			instructorClasses[participantID] = classNumbers
		}

		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(instructorClasses)
		if err != nil {
			return err
		}

		_, err = tx.Put(instructorClassesKey, &blobEntity{Version: m.Version, Data: buf.Bytes()})
		if err != nil {
			return err
		}
		_, err = tx.Put(metaKey, &m)
		return err
	})
	return err
}

func (s *Store) GetPrintSignatures(ctx context.Context) (map[string]string, error) {
	var blob blobEntity
	err := noEntityOK(s.client.Get(ctx, printSignaturesKey, &blob))
	if err != nil {
		return nil, err
	}

	var printSignatures map[string]string
	if len(blob.Data) == 0 {
		printSignatures = make(map[string]string)
	} else {
		err := gob.NewDecoder(bytes.NewReader(blob.Data)).Decode(&printSignatures)
		if err != nil {
			return nil, err
		}
	}

	return printSignatures, nil
}

func (s *Store) SetPrintSignatures(ctx context.Context, modifiedSignatures map[string]string) error {
	_, err := s.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var blob blobEntity
		err := noEntityOK(tx.Get(printSignaturesKey, &blob))
		if err != nil {
			return err
		}

		var printSignatures map[string]string
		if len(blob.Data) == 0 {
			printSignatures = make(map[string]string)
		} else {
			err := gob.NewDecoder(bytes.NewReader(blob.Data)).Decode(&printSignatures)
			if err != nil {
				return err
			}
		}

		for id, sig := range modifiedSignatures {
			if sig == "" {
				delete(printSignatures, id)
			} else {
				printSignatures[id] = sig
			}
		}

		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(printSignatures)
		if err != nil {
			return err
		}

		_, err = tx.Put(printSignaturesKey, &blobEntity{Data: buf.Bytes()})
		return err
	})
	return err
}

func (s *Store) GetEvaluation(ctx context.Context, participantID string) (*conference.Evaluation, error) {
	var blob blobEntity
	err := noEntityOK(s.client.Get(ctx, evaluationKey(participantID), &blob))
	if err != nil {
		return nil, err
	}
	var eval conference.Evaluation
	if len(blob.Data) > 0 {
		err = gob.NewDecoder(bytes.NewReader(blob.Data)).Decode(&eval)
		if err != nil {
			return nil, fmt.Errorf("store.eval: error decoded gob: %w", err)
		}
	}
	eval.ParticipantID = participantID
	return &eval, err
}

func (s *Store) SetEvaluation(ctx context.Context, participantID string, modifiedEval *conference.Evaluation) error {
	key := evaluationKey(participantID)

	_, err := s.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

		var blob blobEntity
		err := noEntityOK(tx.Get(key, &blob))
		if err != nil {
			return err
		}

		var eval conference.Evaluation
		if len(blob.Data) > 0 {
			err = gob.NewDecoder(bytes.NewReader(blob.Data)).Decode(&eval)
			if err != nil {
				return err
			}
		}

		if modifiedEval.Conference != nil {
			eval.Conference = modifiedEval.Conference
		}
		if modifiedEval.Note != nil {
			eval.Note = modifiedEval.Note
		}
		for _, se := range modifiedEval.Sessions {
			eval.SetSession(se)
		}

		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(&eval)
		if err != nil {
			return err
		}

		_, err = tx.Put(key, &blobEntity{Data: buf.Bytes()})
		return err
	})
	return err
}

func (s *Store) DeleteBlob(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("store: empty blob name")
	}
	return noEntityOK(s.client.Delete(ctx, blobKey(name)))
}

func noEntityOK(err error) error {
	if err == datastore.ErrNoSuchEntity {
		return nil
	}
	if errs, ok := err.(datastore.MultiError); ok {
		for _, err := range errs {
			if err != nil && err != datastore.ErrNoSuchEntity {
				return errs
			}
		}
		return nil
	}
	return err
}

func assignLoginCodes(loginCodes map[string]string, participants []*conference.Participant) error {

	assigned := make(map[string]bool)
	for _, code := range loginCodes {
		assigned[code] = true
	}

	var b [4]byte
	for _, p := range participants {
		p.LoginCode = loginCodes[p.ID]
		if p.LoginCode == "" {
			for i := 0; i < 10000; i++ {
				if _, err := rand.Read(b[:]); err != nil {
					return err
				}
				n := int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
				code := strconv.Itoa(n%899999 + 100000)
				if assigned[code] {
					continue
				}
				assigned[code] = true
				loginCodes[p.ID] = code
				p.LoginCode = code
				break
			}
		}
		if p.LoginCode == "" {
			return errors.New("could not assign login code")
		}
	}
	return nil
}
