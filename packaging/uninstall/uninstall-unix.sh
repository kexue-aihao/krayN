#!/usr/bin/env sh
set -eu

INSTALL_DIR=""
ALL_USERS=0
KEEP_PROFILES=0
SKIP_INSTALL_DIR=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --install-dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    --all-users)
      ALL_USERS=1
      shift
      ;;
    --keep-profiles)
      KEEP_PROFILES=1
      shift
      ;;
    --skip-install-dir)
      SKIP_INSTALL_DIR=1
      shift
      ;;
    *)
      shift
      ;;
  esac
done

locale_hint="${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}"
case "$locale_hint" in
  zh*) ZH=1 ;;
  *) ZH=0 ;;
esac

msg() {
  key="$1"
  if [ "$ZH" -eq 1 ]; then
    case "$key" in
      title) printf '%s\n' 'krayN 完整卸载' ;;
      stop) printf '%s\n' '正在停止 krayN 进程...' ;;
      package) printf '%s\n' '正在删除系统安装包...' ;;
      data) printf '%s\n' '正在删除用户配置和缓存...' ;;
      install) printf '%s\n' '正在删除安装目录...' ;;
      keep) printf '%s\n' '已按要求保留用户配置。' ;;
      done) printf '%s\n' 'krayN 已完整卸载。' ;;
      removed) printf '删除：%s\n' "$2" ;;
      warn) printf '警告：%s\n' "$2" ;;
      skip) printf '跳过路径：%s\n' "$2" ;;
    esac
  else
    case "$key" in
      title) printf '%s\n' 'krayN complete uninstall' ;;
      stop) printf '%s\n' 'Stopping krayN processes...' ;;
      package) printf '%s\n' 'Removing system package...' ;;
      data) printf '%s\n' 'Removing user configuration and cache...' ;;
      install) printf '%s\n' 'Removing install directory...' ;;
      keep) printf '%s\n' 'User profiles were kept as requested.' ;;
      done) printf '%s\n' 'krayN has been completely uninstalled.' ;;
      removed) printf 'Removed: %s\n' "$2" ;;
      warn) printf 'Warning: %s\n' "$2" ;;
      skip) printf 'Skipped path: %s\n' "$2" ;;
    esac
  fi
}

run_rm() {
  path="$1"
  [ -n "$path" ] || return 0
  [ -e "$path" ] || return 0
  if rm -rf "$path"; then
    msg removed "$path"
  else
    msg warn "$path"
  fi
}

run_sudo_rm() {
  path="$1"
  [ -n "$path" ] || return 0
  [ -e "$path" ] || return 0
  if [ "$(id -u)" -eq 0 ]; then
    run_rm "$path"
  elif command -v sudo >/dev/null 2>&1; then
    if sudo rm -rf "$path"; then
      msg removed "$path"
    else
      msg warn "$path"
    fi
  else
    msg warn "$path"
  fi
}

safe_install_dir() {
  path="$1"
  [ -n "$path" ] || return 1
  [ -d "$path" ] || return 1
  base="$(basename "$path")"
  case "$base" in
    krayN|krayn|krayn.app|krayN.app) ;;
    *) return 1 ;;
  esac
  [ -e "$path/krayn" ] || [ -e "$path/krayn-core" ] || [ -e "$path/Contents/MacOS/krayn" ] || [ -e "$path/Contents/MacOS/krayn-core" ]
}

remove_user_data() {
  home_dir="$1"
  os_name="$(uname -s)"
  if [ "$os_name" = "Darwin" ]; then
    run_rm "$home_dir/Library/Application Support/krayN"
    run_rm "$home_dir/Library/Caches/krayN"
    run_rm "$home_dir/Library/Saved Application State/io.krayn.krayn.savedState"
    run_rm "$home_dir/Library/Preferences/io.krayn.krayn.plist"
  else
    if [ "$home_dir" = "${HOME:-}" ]; then
      config_home="${XDG_CONFIG_HOME:-$home_dir/.config}"
      cache_home="${XDG_CACHE_HOME:-$home_dir/.cache}"
      data_home="${XDG_DATA_HOME:-$home_dir/.local/share}"
    else
      config_home="$home_dir/.config"
      cache_home="$home_dir/.cache"
      data_home="$home_dir/.local/share"
    fi
    run_rm "$config_home/krayN"
    run_rm "$cache_home/krayN"
    run_rm "$data_home/krayN"
    run_rm "$data_home/krayn"
    run_rm "$data_home/applications/krayN.desktop"
    run_rm "$data_home/icons/hicolor/scalable/apps/krayN.svg"
  fi
}

msg title
msg stop
pkill -x krayn 2>/dev/null || true
pkill -x krayn-core 2>/dev/null || true

if [ "$(uname -s)" = "Linux" ]; then
  msg package
  if command -v dpkg >/dev/null 2>&1 && dpkg -s krayn >/dev/null 2>&1; then
    if [ "$(id -u)" -eq 0 ]; then
      dpkg -r krayn || true
    elif command -v sudo >/dev/null 2>&1; then
      sudo dpkg -r krayn || true
    fi
  elif command -v rpm >/dev/null 2>&1 && rpm -q krayn >/dev/null 2>&1; then
    if [ "$(id -u)" -eq 0 ]; then
      rpm -e krayn || true
    elif command -v sudo >/dev/null 2>&1; then
      sudo rpm -e krayn || true
    fi
  fi
fi

if [ "$KEEP_PROFILES" -eq 1 ]; then
  msg keep
else
  msg data
  if [ "$ALL_USERS" -eq 1 ] && [ "$(id -u)" -eq 0 ]; then
    for home_dir in /home/* /Users/*; do
      [ -d "$home_dir" ] || continue
      remove_user_data "$home_dir"
    done
  else
    remove_user_data "$HOME"
  fi
fi

if [ "$SKIP_INSTALL_DIR" -eq 0 ]; then
  msg install
  for dir in "$INSTALL_DIR" /opt/krayN /Applications/krayn.app /Applications/krayN.app "$HOME/Applications/krayn.app" "$HOME/Applications/krayN.app"; do
    [ -n "$dir" ] || continue
    if safe_install_dir "$dir"; then
      case "$dir" in
        /opt/*|/Applications/*) run_sudo_rm "$dir" ;;
        *) run_rm "$dir" ;;
      esac
    elif [ -e "$dir" ]; then
      msg skip "$dir"
    fi
  done
fi

msg done
