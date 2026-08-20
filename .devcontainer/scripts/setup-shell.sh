#!/usr/bin/env bash
set -Eeuo pipefail

BASHRC="/root/.bashrc"
HIST_DIR="/root/.history"
HIST_FILE="${HIST_DIR}/.bash_history"

replace_block() {
    local begin="$1"
    local end="$2"
    local file="$3"

    if grep -qF "$begin" "$file" 2>/dev/null; then
        awk -v start="$begin" -v finish="$end" '
            $0 == start { in_block = 1; next }
            $0 == finish { in_block = 0; next }
            !in_block { print }
        ' "$file" >"${file}.tmp"
        mv "${file}.tmp" "$file"
    fi
}

mkdir -p "$HIST_DIR" /root/.config /etc/bash_completion.d
touch "$HIST_FILE"
chmod 700 "$HIST_DIR"
chmod 600 "$HIST_FILE"
ln -sf "$HIST_FILE" /root/.bash_history

HIST_BEGIN="# >>> switchonyourcode: persistent bash history begin >>>"
HIST_END="# <<< switchonyourcode: persistent bash history end <<<"
replace_block "$HIST_BEGIN" "$HIST_END" "$BASHRC"

cat >>"$BASHRC" <<EOF
${HIST_BEGIN}
export HISTFILE="${HIST_FILE}"
export HISTSIZE=50000
export HISTFILESIZE=100000
export HISTCONTROL=ignoredups:erasedups
export HISTTIMEFORMAT='%F %T '
shopt -s histappend
PROMPT_COMMAND="history -a; history -n; \${PROMPT_COMMAND:-}"
${HIST_END}
EOF

COMP_BEGIN="# >>> switchonyourcode: bash completion begin >>>"
COMP_END="# <<< switchonyourcode: bash completion end <<<"
replace_block "$COMP_BEGIN" "$COMP_END" "$BASHRC"

cat >>"$BASHRC" <<'EOF'
# >>> switchonyourcode: bash completion begin >>>
if [ -n "$PS1" ]; then
    if [ -r /etc/profile.d/bash_completion.sh ]; then
        . /etc/profile.d/bash_completion.sh
    elif [ -f /usr/share/bash-completion/bash_completion ]; then
        . /usr/share/bash-completion/bash_completion
    fi
fi
bind "set completion-ignore-case on"
bind "set show-all-if-ambiguous on"
bind "set menu-complete-display-prefix on"
# <<< switchonyourcode: bash completion end <<<
EOF

if command -v gh >/dev/null 2>&1; then
    gh completion -s bash >/etc/bash_completion.d/gh
fi

if command -v npm >/dev/null 2>&1; then
    npm completion >/etc/bash_completion.d/npm
    cp -f /etc/bash_completion.d/npm /etc/bash_completion.d/npx
fi

if command -v pnpm >/dev/null 2>&1; then
    pnpm completion bash >/etc/bash_completion.d/pnpm
fi

if command -v docker >/dev/null 2>&1; then
    docker completion bash >/etc/bash_completion.d/docker 2>/dev/null || true
fi

STARSHIP_BEGIN="# >>> switchonyourcode: starship init begin >>>"
STARSHIP_END="# <<< switchonyourcode: starship init end <<<"
replace_block "$STARSHIP_BEGIN" "$STARSHIP_END" "$BASHRC"

cat >>"$BASHRC" <<'EOF'
# >>> switchonyourcode: starship init begin >>>
if command -v starship >/dev/null 2>&1; then
    eval "$(starship init bash)"
fi
# <<< switchonyourcode: starship init end <<<
EOF

cat >/root/.config/starship.toml <<'EOF'
add_newline = false
format = "$directory$git_branch$git_status$golang$nodejs$cmd_duration$character"

[directory]
truncation_length = 3
truncate_to_repo = true

[git_status]
disabled = true

[cmd_duration]
min_time = 1000

[character]
success_symbol = "\\$ "
error_symbol = "! "
EOF
