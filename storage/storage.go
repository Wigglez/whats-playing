package storage

import (
	"fmt"
	"os"

	bolt "go.etcd.io/bbolt"
)

type Storage struct {
	DB *bolt.DB
}

func (s Storage) Get(bucket []byte, record []byte) []byte {
	var data = []byte(nil)

	err := s.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		v := b.Get(record)
		if v != nil {
			data = v
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error reading data: %v", err.Error())
	}
	return data
}

func (s Storage) Set(bucket []byte, record []byte, data []byte) {
	err := s.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		err := b.Put(record, data)
		return err
	})
	if err != nil {
		fmt.Printf("Error writing data: %v", err.Error())
	}
}

func (s *Storage) Open() {
	base_dir, err := os.UserConfigDir()
	separator := string(os.PathSeparator)
	if err == nil {
		os.Mkdir(fmt.Sprintf("%s%swhatsplaying", base_dir, separator), 0755)
		fmt.Println(os.UserConfigDir())
		var err error
		s.DB, err = bolt.Open(fmt.Sprintf("%s%swhatsplaying%sstorage.db", base_dir, separator, separator), 0600, nil)
		if err != nil {
			fmt.Printf("Error opening storage")
		}

		s.DB.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte("plex-token"))
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
			return nil
		})
		s.DB.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte("imgur-urls"))
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
			return nil
		})
	} else {
		fmt.Println("Error setting up storage")
	}
}
