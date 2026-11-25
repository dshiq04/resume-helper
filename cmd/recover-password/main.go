// Command recover-password interactively generates a new application password
// for a ResumeTrail database. It is intentionally kept outside the desktop
// binary so a forgotten password can be recovered from a terminal.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"resume-helper/internal/config"
	"resume-helper/internal/database"
	"resume-helper/internal/models"
	"resume-helper/internal/security"

	"gorm.io/gorm"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("履途 ResumeTrail - 恢复密码 / Password recovery")
	fmt.Println(strings.Repeat("=", 52))
	fmt.Println("本工具会连接数据库并重置应用登录密码。")
	fmt.Println("This tool connects to the database and resets the app password.")
	fmt.Println()

	cfg, _ := config.Load()
	defaults := database.Settings{Driver: "sqlite"}
	if cfg != nil {
		defaults = database.ParseSettings(cfg.Database.Driver, cfg.Database.DSN, cfg.DataDir)
	}

	path := prompt(reader, "SQLite 文件路径 / SQLite file", defaults.SQLitePath)
	if path == "" {
		path = "resume-helper.db"
	}

	fmt.Println()
	fmt.Printf("即将连接 / Connecting to: %s\n", path)
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: path})
	if err != nil {
		fail(fmt.Errorf("连接数据库失败 / cannot connect: %w", err))
	}

	var existing models.Setting
	// The recovery flow only makes sense when a password has actually been set.
	// If the settings table has no password key, stop instead of silently
	// creating one (that would turn an unprotected profile into a locked one).
	if err := db.First(&existing, "key = ?", "auth.password_hash").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(fmt.Errorf("设置表中没有找到密码键 auth.password_hash，已停止恢复 / no password key found in the settings table"))
		}
		fail(fmt.Errorf("读取密码设置失败 / read password setting: %w", err))
	}
	if strings.TrimSpace(existing.Value) == "" {
		fail(fmt.Errorf("密码键 auth.password_hash 为空，已停止恢复 / the password key is empty"))
	}
	fmt.Println("检测到已存在的密码，将继续重置。")
	fmt.Println("An existing password was found and will be replaced.")

	if !confirm(reader, "继续生成新密码? / Generate a new password? (y/N)", false) {
		fmt.Println("已取消 / Cancelled.")
		return
	}

	password, err := security.GeneratePassword(security.RecoveryPasswordLength)
	if err != nil {
		fail(fmt.Errorf("生成密码失败 / generate password: %w", err))
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		fail(fmt.Errorf("写入密码失败 / hash password: %w", err))
	}
	if err := db.Save(&models.Setting{Key: "auth.password_hash", Value: hash}).Error; err != nil {
		fail(fmt.Errorf("保存密码失败 / save password: %w", err))
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 52))
	fmt.Println("密码已重置 / Password reset successfully")
	fmt.Printf("新密码 / New password: %s\n", password)
	fmt.Println(strings.Repeat("-", 52))
	fmt.Println("请立即使用该密码登录，并在登录后到设置中修改密码。")
	fmt.Println("Sign in with this password and change it in Settings afterwards.")
}

func prompt(reader *bufio.Reader, label, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		fmt.Printf("%s [%s]: ", label, fallback)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return strings.TrimSpace(fallback)
	}
	return line
}

func confirm(reader *bufio.Reader, label string, fallback bool) bool {
	suffix := "y/N"
	if fallback {
		suffix = "Y/n"
	}
	fmt.Printf("%s [%s]: ", label, suffix)
	line, _ := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return fallback
	}
	return line == "y" || line == "yes" || line == "是"
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "\n错误 / Error: %v\n", err)
	os.Exit(1)
}
