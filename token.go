package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path"

	"golang.org/x/oauth2"
)

type TokenManager struct {
	token       *oauth2.Token
	persistPath string
}

func (tm *TokenManager) SetToken(token *oauth2.Token) error {
	tm.token = token
	return tm.saveToken(tm.persistPath)
}

func (tm *TokenManager) Auth() error {
	if tm.token != nil {
		return tm.saveToken(tm.persistPath)
	}

	token, err := tm.loadToken()
	if err != nil {
		return err
	}

	tm.token = token
	return nil
}

func (tm *TokenManager) saveToken(p string) error {
	if p == "" {
		return fmt.Errorf("empty path")
	}

	dir := path.Dir(tm.persistPath)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	file, err := os.Create(p)
	if err != nil {
		return err
	}

	if tm.token == nil {
		return fmt.Errorf("token is nil")
	}

	return gob.NewEncoder(file).Encode(tm.token)
}

func (tm *TokenManager) loadToken() (*oauth2.Token, error) {
	if tm.persistPath == "" {
		return nil, fmt.Errorf("empty persisten token path")
	}

	file, err := os.Open(tm.persistPath)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	err = gob.NewDecoder(file).Decode(&token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}
