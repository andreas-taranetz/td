PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
PATH_LINE := export PATH="$(BINDIR):$$PATH"

.PHONY: install

install:
	mkdir -p "$(BINDIR)"
	go build -o "$(BINDIR)/td" .
	@if printf '%s' ":$$PATH:" | grep -Fq ":$(BINDIR):"; then \
		printf 'Installed td to %s/td\n' "$(BINDIR)"; \
		printf '%s is already on PATH\n' "$(BINDIR)"; \
	else \
		shell_name=$$(basename "$${SHELL:-}"); \
		case "$$shell_name" in \
			zsh) rc_file="$$HOME/.zshrc" ;; \
			bash) rc_file="$$HOME/.bashrc" ;; \
			*) rc_file="" ;; \
		esac; \
		printf 'Installed td to %s/td\n' "$(BINDIR)"; \
		printf '%s is not on PATH\n' "$(BINDIR)"; \
		if [ -n "$$rc_file" ]; then \
			printf 'Append PATH update to %s? [y/N] ' "$$rc_file"; \
			read answer; \
			case "$$answer" in \
				[yY]|[yY][eE][sS]) \
					if [ -f "$$rc_file" ] && grep -Fq '$(BINDIR)' "$$rc_file"; then \
						printf '%s already references %s\n' "$$rc_file" "$(BINDIR)"; \
					else \
						printf '\n%s\n' '$(PATH_LINE)' >> "$$rc_file"; \
						printf 'Updated %s\n' "$$rc_file"; \
					fi; \
					printf 'Reload your shell with: source %s\n' "$$rc_file"; \
					;; \
				*) \
					printf 'Add this line manually to %s:\n%s\n' "$$rc_file" '$(PATH_LINE)'; \
					;; \
			esac; \
		else \
			printf 'Could not determine a supported shell config file from SHELL=%s\n' "$${SHELL:-unknown}"; \
			printf 'Add this line to your shell config:\n%s\n' '$(PATH_LINE)'; \
		fi; \
	fi
