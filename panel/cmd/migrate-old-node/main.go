// Command migrate-old-node is a one-off tool: it reads users out of an old
// single-node rookeryd's SQLite database (id, name, secret — the pre-panel
// schema) and recreates each as a panel user with one subscription, assigned
// to a given node. The old per-user secret has no equivalent in the new
// token-based auth model, so every migrated user needs the newly printed
// sub_url — their old rookery:// link stops working.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate-old-node:", err)
		os.Exit(1)
	}
}

func run() error {
	oldDBPath := flag.String("old-db", "", "path to the old node's rookery.db")
	panelAddr := flag.String("panel-addr", "", "panel base URL, e.g. https://panel.example.com")
	adminUser := flag.String("admin-user", "", "panel admin username")
	adminPass := flag.String("admin-pass", "", "panel admin password")
	nodeID := flag.String("node-id", "", "node ID (from the panel's Nodes tab) to assign every migrated subscription to")
	flag.Parse()

	if *oldDBPath == "" || *panelAddr == "" || *adminUser == "" || *adminPass == "" || *nodeID == "" {
		flag.Usage()
		return fmt.Errorf("all flags are required")
	}

	users, err := readOldUsers(*oldDBPath)
	if err != nil {
		return fmt.Errorf("read old db: %w", err)
	}
	if len(users) == 0 {
		fmt.Println("no users found in old db — nothing to migrate")
		return nil
	}
	fmt.Printf("found %d user(s) in old db\n\n", len(users))

	c, err := newPanelClient(*panelAddr, *adminUser, *adminPass)
	if err != nil {
		return fmt.Errorf("log in to panel: %w", err)
	}

	for _, u := range users {
		newUser, err := c.createUser(u.Name)
		if err != nil {
			fmt.Printf("[%s] FAILED to create user: %v\n", u.Name, err)
			continue
		}
		sub, err := c.createSubscription(newUser.ID, u.Name)
		if err != nil {
			fmt.Printf("[%s] FAILED to create subscription: %v\n", u.Name, err)
			continue
		}
		if err := c.setSubscriptionNodes(sub.ID, []string{*nodeID}); err != nil {
			fmt.Printf("[%s] FAILED to assign node: %v\n", u.Name, err)
			continue
		}
		fmt.Printf("[%s] migrated -> %s\n", u.Name, sub.SubURL)
	}

	fmt.Println("\nSend each printed sub_url to the corresponding user — their old rookery:// link no longer works.")
	return nil
}

type oldUser struct {
	ID   string
	Name string
}

func readOldUsers(path string) ([]oldUser, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []oldUser
	for rows.Next() {
		var u oldUser
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

type panelClient struct {
	baseAddr string
	http     *http.Client
}

func newPanelClient(baseAddr, username, password string) (*panelClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &panelClient{baseAddr: baseAddr, http: &http.Client{Jar: jar}}

	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := c.http.Post(baseAddr+"/admin/api/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("login: panel returned status %d", resp.StatusCode)
	}
	return c, nil
}

type panelUser struct {
	ID string `json:"id"`
}

func (c *panelClient) createUser(name string) (panelUser, error) {
	var u panelUser
	err := c.postJSON("/admin/api/users", map[string]string{"name": name}, &u)
	return u, err
}

type panelSubscription struct {
	ID     string `json:"id"`
	SubURL string `json:"sub_url"`
}

func (c *panelClient) createSubscription(userID, name string) (panelSubscription, error) {
	var s panelSubscription
	err := c.postJSON("/admin/api/subscriptions", map[string]string{"user_id": userID, "name": name}, &s)
	return s, err
}

func (c *panelClient) setSubscriptionNodes(subID string, nodeIDs []string) error {
	body, _ := json.Marshal(map[string][]string{"node_ids": nodeIDs})
	req, err := http.NewRequest(http.MethodPut, c.baseAddr+"/admin/api/subscriptions/"+subID+"/nodes", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("panel returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *panelClient) postJSON(path string, reqBody, respBody any) error {
	data, _ := json.Marshal(reqBody)
	resp, err := c.http.Post(c.baseAddr+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("panel returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(respBody)
}
