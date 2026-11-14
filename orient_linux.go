//go:build linux

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const orientKey = "/lipstick/orientationLock"

func orientStampPath() string {
	dir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if dir == "" {
		dir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return filepath.Join(dir, "cargadgetd-link-orient")
}

func sessionBusEnv() []string {
	env := os.Environ()
	if strings.TrimSpace(os.Getenv("DBUS_SESSION_BUS_ADDRESS")) != "" {
		return env
	}
	candidates := []string{
		fmt.Sprintf("/run/user/%d/dbus/user_bus_socket", os.Getuid()),
		"/run/user/100000/dbus/user_bus_socket",
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		candidates = append([]string{xdg + "/dbus/user_bus_socket"}, candidates...)
	}
	for _, sock := range candidates {
		if _, err := os.Stat(sock); err == nil {
			return append(env, "DBUS_SESSION_BUS_ADDRESS=unix:path="+sock)
		}
	}
	return env
}

func dconf(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "dconf", args...)
	cmd.Env = sessionBusEnv()
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("dconf timed out")
	}
	if err != nil {
		return string(out), fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func readOrient() (string, error) {
	out, err := dconf("read", orientKey)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		return "dynamic", nil
	}
	v, ok := parseOrient(out)
	if !ok {
		return "", fmt.Errorf("orientation %q", strings.TrimSpace(out))
	}
	return v, nil
}

func writeOrient(v string) error {
	_, err := dconf("write", orientKey, "'"+v+"'")
	return err
}

func readStamp() (string, error) {
	b, err := os.ReadFile(orientStampPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	v, ok := parseOrient(string(b))
	if !ok {
		return "", fmt.Errorf("stamp %q", strings.TrimSpace(string(b)))
	}
	return v, nil
}

func writeStamp(v string) error {
	return os.WriteFile(orientStampPath(), []byte(v+"\n"), 0o644)
}

func removeStamp() {
	_ = os.Remove(orientStampPath())
}

// lockPhoneLandscape pins the phone UI to landscape for the whole time Link
// is on screen. The previous lock is stamped so a killed MD can restore it.
func lockPhoneLandscape() {
	prev, err := readStamp()
	if err != nil {
		log.Printf("link: orientation stamp: %v", err)
		prev = ""
	}
	cur, err := readOrient()
	if err != nil {
		log.Printf("link: orientation read: %v", err)
		return
	}
	if cur == orientLandscape && prev == "" {
		return
	}
	if prev == "" {
		if err := writeStamp(cur); err != nil {
			log.Printf("link: orientation stamp: %v", err)
			return
		}
		prev = cur
	}
	if cur == orientLandscape {
		return
	}
	if err := writeOrient(orientLandscape); err != nil {
		log.Printf("link: orientation lock: %v", err)
		return
	}
	log.Printf("link: orientation locked landscape (was %s)", prev)
}

func unlockPhoneLandscape() {
	prev, err := readStamp()
	if err != nil {
		log.Printf("link: orientation stamp: %v", err)
		return
	}
	if prev == "" {
		return
	}
	if err := writeOrient(prev); err != nil {
		log.Printf("link: orientation restore: %v", err)
		return
	}
	removeStamp()
	log.Printf("link: orientation restored %s", prev)
}
