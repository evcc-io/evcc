package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/jlaffaye/ftp"
)

func runFTPBackupRoutine(stopC <-chan struct{}, conf globalconfig.FTPBackup) {
	if strings.TrimSpace(conf.Host) == "" {
		return
	}

	if conf.Port == 0 {
		conf.Port = 21
	}

	if conf.Schedule == "" {
		conf.Schedule = "03:00"
	}

	if conf.Timeout <= 0 {
		conf.Timeout = 30 * time.Second
	}

	log.INFO.Printf("FTP backup enabled: host=%s, directory=%s, schedule=%s", conf.Host, conf.Directory, conf.Schedule)

	go func() {
		for {
			runAt, err := nextDailyRun(time.Now(), conf.Schedule)
			if err != nil {
				log.ERROR.Printf("ftp backup disabled due to invalid schedule %q: %v", conf.Schedule, err)
				return
			}

			timer := time.NewTimer(time.Until(runAt))
			select {
			case <-stopC:
				timer.Stop()
				return
			case <-timer.C:
				if err := uploadBackup(conf); err != nil {
					log.ERROR.Printf("ftp backup failed: %v", err)
				} else {
					log.INFO.Println("ftp backup completed successfully")
				}
			}
		}
	}()
}

func nextDailyRun(now time.Time, schedule string) (time.Time, error) {
	parts := strings.Split(schedule, ":")
	if len(parts) != 2 {
		return time.Time{}, errors.New("expected format HH:MM")
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return time.Time{}, errors.New("hour must be between 00 and 23")
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return time.Time{}, errors.New("minute must be between 00 and 59")
	}

	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}

	return next, nil
}

func uploadBackup(conf globalconfig.FTPBackup) error {
	if err := settings.Persist(); err != nil {
		log.WARN.Printf("unable to persist settings before ftp backup: %v", err)
	}

	file, err := os.Open(db.FilePath)
	if err != nil {
		return fmt.Errorf("open database file: %w", err)
	}
	defer file.Close()

	addr := net.JoinHostPort(conf.Host, strconv.Itoa(conf.Port))

	timeout, err := time.ParseDuration(conf.Timeout)
	if conf.Timeout == "" || err != nil || timeout <= 0 {
		timeout = 30 * time.Second
	}

	options := []ftp.DialOption{ftp.DialWithTimeout(timeout)}
	if conf.TLS {
		options = append(options, ftp.DialWithExplicitTLS(&tls.Config{
			ServerName:         conf.Host,
			InsecureSkipVerify: conf.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}))
	}

	client, err := ftp.Dial(addr, options...)
	if err != nil {
		return fmt.Errorf("connect to ftp server: %w", err)
	}
	defer client.Quit()

	if err := client.Login(conf.User, conf.Password); err != nil {
		return fmt.Errorf("ftp login failed: %w", err)
	}

	remoteName := "evcc-backup-" + time.Now().Format("2006-01-02--15-04") + ".db"
	remotePath := path.Join("/", strings.TrimSpace(conf.Directory), remoteName)

	if err := client.Stor(remotePath, file); err != nil {
		return fmt.Errorf("upload backup to %s: %w", remotePath, err)
	}

	return nil
}
