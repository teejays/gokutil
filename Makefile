
.PHONY: gokutil-mod-upgrade

# Color Control Sequences for easy printing
_RESET=\033[0m
_RED=\033[31;1m
_GREEN=\033[32;1m
_YELLOW=\033[33;1m
_BLUE=\033[34;1m
_MAGENTA=\033[35;1m
_CYAN=\033[36;1m
_WHITE=\033[37;1m

# Loop through all the dirs in ./gokutil and if there is a go.mod file, run go mod upgrade
go-mod-upgrade:
	@for dir in $(shell ls -d ./*/); do \
		if [ -f $$dir/go.mod ]; then \
			echo "Upgrading $$dir"; \
			cd $$dir; \
			go get -u; \
			go mod tidy; \
			cd -; \
		fi; \
	done && \
	git commit -a -m "go mod upgrade" && \
	git push
	