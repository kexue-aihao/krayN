//go:build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

var (
	version = "dev"
	commit  = "unknown"
)

type options struct {
	InstallDir     string
	AllUsers       bool
	KeepProfiles   bool
	SkipInstallDir bool
	Silent         bool
}

type messages struct {
	Title        string
	Stop         string
	Shortcuts    string
	Data         string
	InstallFiles string
	Install      string
	Keep         string
	Done         string
	Skip         string
	Remove       string
	Warn         string
	Schedule     string
}

func main() {
	opts := parseOptions(os.Args[1:])
	msg := localizedMessages()

	writeLine(opts, msg.Title)
	writeLine(opts, msg.Stop)
	stopKrayNProcesses()

	writeLine(opts, msg.Shortcuts)
	for _, path := range shortcutPaths(opts.AllUsers) {
		removePathIfExists(path, opts, msg)
	}

	if opts.KeepProfiles {
		writeLine(opts, msg.Keep)
	} else {
		writeLine(opts, msg.Data)
		for _, profileRoot := range profileRoots(opts.AllUsers) {
			for _, path := range profileDataPaths(profileRoot) {
				removePathIfExists(path, opts, msg)
			}
		}
	}

	candidates := candidateInstallDirs(opts.InstallDir)
	safeDirs := make([]string, 0, len(candidates))
	for _, dir := range candidates {
		if isSafeInstallDir(dir) {
			safeDirs = append(safeDirs, cleanPath(dir))
		} else if strings.TrimSpace(dir) != "" {
			writef(opts, msg.Skip, dir)
		}
	}
	safeDirs = uniquePaths(safeDirs)

	writeLine(opts, msg.InstallFiles)
	for _, dir := range safeDirs {
		clearInstallDirContents(dir, opts, msg)
	}

	if !opts.SkipInstallDir {
		writeLine(opts, msg.Install)
		for _, dir := range safeDirs {
			removePathIfExists(dir, opts, msg)
			if executableInsideDir(dir) && pathExists(dir) {
				writef(opts, msg.Schedule, dir)
				scheduleDirRemoval(dir)
			}
		}
	}

	writeLine(opts, msg.Done)
}

func parseOptions(args []string) options {
	var opts options
	var installDirAlias string
	var allUsersAlias bool
	var keepProfilesAlias bool
	var skipInstallDirAlias bool
	var silentAlias bool

	flags := flag.NewFlagSet("krayn-uninstall", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&opts.InstallDir, "install-dir", "", "")
	flags.StringVar(&installDirAlias, "InstallDir", "", "")
	flags.BoolVar(&opts.AllUsers, "all-users", false, "")
	flags.BoolVar(&allUsersAlias, "AllUsers", false, "")
	flags.BoolVar(&opts.KeepProfiles, "keep-profiles", false, "")
	flags.BoolVar(&keepProfilesAlias, "KeepProfiles", false, "")
	flags.BoolVar(&opts.SkipInstallDir, "skip-install-dir", false, "")
	flags.BoolVar(&skipInstallDirAlias, "SkipInstallDir", false, "")
	flags.BoolVar(&opts.Silent, "silent", false, "")
	flags.BoolVar(&silentAlias, "Silent", false, "")
	_ = flags.Parse(normalizeArgs(args))

	if opts.InstallDir == "" {
		opts.InstallDir = installDirAlias
	}
	opts.AllUsers = opts.AllUsers || allUsersAlias
	opts.KeepProfiles = opts.KeepProfiles || keepProfilesAlias
	opts.SkipInstallDir = opts.SkipInstallDir || skipInstallDirAlias
	opts.Silent = opts.Silent || silentAlias
	return opts
}

func normalizeArgs(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, "/") && !strings.HasPrefix(arg, "//") {
			out[i] = "-" + strings.TrimPrefix(arg, "/")
			continue
		}
		out[i] = arg
	}
	return out
}

func localizedMessages() messages {
	if isChineseUI() {
		return messages{
			Title:        "krayN \u5b8c\u6574\u5378\u8f7d",
			Stop:         "\u6b63\u5728\u505c\u6b62 krayN \u8fdb\u7a0b...",
			Shortcuts:    "\u6b63\u5728\u5220\u9664\u5feb\u6377\u65b9\u5f0f...",
			Data:         "\u6b63\u5728\u5220\u9664\u7528\u6237\u914d\u7f6e\u548c\u7f13\u5b58...",
			InstallFiles: "\u6b63\u5728\u6e05\u7406\u5b89\u88c5\u6587\u4ef6...",
			Install:      "\u6b63\u5728\u5220\u9664\u5b89\u88c5\u76ee\u5f55...",
			Keep:         "\u5df2\u6309\u8981\u6c42\u4fdd\u7559\u7528\u6237\u914d\u7f6e\u3002",
			Done:         "krayN \u5df2\u5b8c\u6574\u5378\u8f7d\u3002",
			Skip:         "\u8df3\u8fc7\u8def\u5f84: %s",
			Remove:       "\u5220\u9664: %s",
			Warn:         "\u8b66\u544a: %s",
			Schedule:     "\u5df2\u5b89\u6392\u9000\u51fa\u540e\u5220\u9664: %s",
		}
	}
	return messages{
		Title:        fmt.Sprintf("krayN complete uninstall %s (%s)", version, commit),
		Stop:         "Stopping krayN processes...",
		Shortcuts:    "Removing shortcuts...",
		Data:         "Removing user configuration and cache...",
		InstallFiles: "Removing installed files...",
		Install:      "Removing install directory...",
		Keep:         "User profiles were kept as requested.",
		Done:         "krayN has been completely uninstalled.",
		Skip:         "Skipped path: %s",
		Remove:       "Removed: %s",
		Warn:         "Warning: %s",
		Schedule:     "Scheduled deletion after exit: %s",
	}
}

func isChineseUI() bool {
	if primaryLanguageID(getUserDefaultUILanguage()) == 0x04 {
		return true
	}
	for _, key := range []string{"LANG", "LC_ALL", "LC_MESSAGES"} {
		if strings.HasPrefix(strings.ToLower(os.Getenv(key)), "zh") {
			return true
		}
	}
	return false
}

func getUserDefaultUILanguage() uint16 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultUILanguage")
	value, _, err := proc.Call()
	if value == 0 && err != syscall.Errno(0) {
		return 0
	}
	return uint16(value)
}

func primaryLanguageID(langID uint16) uint16 {
	return langID & 0x03ff
}

func stopKrayNProcesses() {
	for _, image := range []string{"krayn.exe", "krayn-core.exe"} {
		runHidden("taskkill.exe", "/T", "/IM", image)
	}
	time.Sleep(800 * time.Millisecond)
	for _, image := range []string{"krayn.exe", "krayn-core.exe"} {
		runHidden("taskkill.exe", "/F", "/T", "/IM", image)
	}
	time.Sleep(500 * time.Millisecond)
}

func runHidden(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}

func shortcutPaths(allUsers bool) []string {
	paths := []string{
		filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "krayN"),
		filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu", "Programs", "krayN"),
		filepath.Join(os.Getenv("USERPROFILE"), "Desktop", "krayN.lnk"),
		filepath.Join(os.Getenv("PUBLIC"), "Desktop", "krayN.lnk"),
	}
	if allUsers {
		for _, root := range profileRoots(true) {
			paths = append(paths, filepath.Join(root, "Desktop", "krayN.lnk"))
		}
	}
	return uniquePaths(paths)
}

func profileRoots(allUsers bool) []string {
	if !allUsers {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			return []string{home}
		}
		return []string{os.Getenv("USERPROFILE")}
	}
	usersRoot := filepath.Join(os.Getenv("SystemDrive")+string(os.PathSeparator), "Users")
	entries, err := os.ReadDir(usersRoot)
	if err != nil {
		return profileRoots(false)
	}
	excluded := map[string]bool{
		"all users":    true,
		"default":      true,
		"default user": true,
		"defaultuser0": true,
		"public":       true,
	}
	roots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || excluded[strings.ToLower(entry.Name())] {
			continue
		}
		roots = append(roots, filepath.Join(usersRoot, entry.Name()))
	}
	if len(roots) == 0 {
		return profileRoots(false)
	}
	return uniquePaths(roots)
}

func profileDataPaths(profileRoot string) []string {
	if strings.TrimSpace(profileRoot) == "" {
		return nil
	}
	return []string{
		filepath.Join(profileRoot, "AppData", "Roaming", "krayN"),
		filepath.Join(profileRoot, "AppData", "Local", "krayN"),
		filepath.Join(profileRoot, "AppData", "Local", "krayn"),
		filepath.Join(profileRoot, "AppData", "Local", "io.krayn.krayn"),
	}
}

func candidateInstallDirs(installDir string) []string {
	paths := []string{installDir}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Dir(exe))
	}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		paths = append(paths, filepath.Join(programFiles, "krayN"))
	}
	if programFilesX86 := os.Getenv("ProgramFiles(x86)"); programFilesX86 != "" {
		paths = append(paths, filepath.Join(programFilesX86, "krayN"))
	}
	return uniquePaths(paths)
}

func isSafeInstallDir(path string) bool {
	cleaned := cleanPath(path)
	if cleaned == "" || !pathExists(cleaned) {
		return false
	}
	leaf := strings.ToLower(filepath.Base(cleaned))
	if !strings.Contains(leaf, "krayn") {
		return false
	}
	for _, marker := range []string{
		"krayn.exe",
		"krayn-core.exe",
		"krayn-uninstall.exe",
		"unins000.exe",
		"uninstall-krayN.ps1",
	} {
		if pathExists(filepath.Join(cleaned, marker)) {
			return true
		}
	}
	return false
}

func clearInstallDirContents(path string, opts options, msg messages) {
	for _, item := range []string{
		"krayn.exe",
		"krayn-core.exe",
		"krayn-uninstall.exe",
		"uninstall-krayN.ps1",
		"flutter_windows.dll",
		"screen_retriever_windows_plugin.dll",
		"tray_manager_plugin.dll",
		"window_manager_plugin.dll",
		"data",
	} {
		removePathIfExists(filepath.Join(path, item), opts, msg)
	}
}

func removePathIfExists(path string, opts options, msg messages) {
	cleaned := cleanPath(path)
	if cleaned == "" || !pathExists(cleaned) {
		return
	}
	makeWritable(cleaned)
	if err := os.RemoveAll(cleaned); err != nil {
		writef(opts, msg.Warn, err.Error())
		return
	}
	writef(opts, msg.Remove, cleaned)
}

func makeWritable(path string) {
	_ = filepath.WalkDir(path, func(current string, _ fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		_ = os.Chmod(current, 0o700)
		return nil
	})
	_ = os.Chmod(path, 0o700)
}

func executableInsideDir(dir string) bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	dir = cleanPath(dir)
	exe = cleanPath(exe)
	rel, err := filepath.Rel(dir, exe)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")
}

func scheduleDirRemoval(dir string) {
	comspec := os.Getenv("ComSpec")
	if comspec == "" {
		comspec = "cmd.exe"
	}
	command := "timeout /T 2 /NOBREAK >NUL & rmdir /S /Q " + cmdQuote(dir)
	cmd := exec.Command(comspec, "/C", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}

func cmdQuote(path string) string {
	return `"` + strings.ReplaceAll(path, `"`, `""`) + `"`
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

func cleanPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return filepath.Clean(trimmed)
	}
	return filepath.Clean(abs)
}

func uniquePaths(paths []string) []string {
	seen := map[string]string{}
	for _, path := range paths {
		cleaned := cleanPath(path)
		if cleaned == "" {
			continue
		}
		seen[strings.ToLower(cleaned)] = cleaned
	}
	out := make([]string, 0, len(seen))
	for _, path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func writeLine(opts options, text string) {
	if !opts.Silent {
		fmt.Println(text)
	}
}

func writef(opts options, format string, args ...any) {
	if !opts.Silent {
		fmt.Printf(format+"\n", args...)
	}
}
