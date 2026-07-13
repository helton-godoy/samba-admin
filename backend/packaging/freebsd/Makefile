PREFIX?=/usr/local
PACKAGE_ROOT!=pwd -P
STAGEDIR=${PACKAGE_ROOT}/stage
BIN_DIR?=../../bin
PKG?=pkg
VERSION?=0.3.5
RELEASE?=no
PACKAGE_ORIGIN?=local/samba-admin
PACKAGE_MAINTAINER?=infraestrutura@exemplo.invalid
PACKAGE_WWW?=https://example.invalid/samba-admin
.if !defined(ARCH)
ARCH!=${PKG} config ABI
.endif
PACKAGE_DIR=${PACKAGE_ROOT}/packages

BINDIR=$(PREFIX)/sbin
LIBEXECDIR=$(PREFIX)/libexec
ETCDIR=$(PREFIX)/etc/samba-admin
RCDIR=$(PREFIX)/etc/rc.d

.PHONY: check safe-stage stage package

safe-stage:
	@test "$(STAGEDIR)" = "$(PACKAGE_ROOT)/stage" || (echo "diretorio de staging inesperado: $(STAGEDIR)" >&2; exit 1)
	@test ! -e "$(STAGEDIR)" && test ! -L "$(STAGEDIR)" || (echo "staging ja existe; revise e remova-o manualmente: $(STAGEDIR)" >&2; exit 1)
	@test ! -e "$(PACKAGE_DIR)/SHA256" || (echo "checksum anterior existe; preserve ou remova-o manualmente" >&2; exit 1)
	@test -z "$$(find "$(PACKAGE_DIR)" -type f -name '*.pkg' -print -quit 2>/dev/null)" || (echo "pacote anterior existe; preserve ou remova-o manualmente" >&2; exit 1)

check:
	@test -x "$(BIN_DIR)/samba-admin-api" || (echo "executavel ausente: $(BIN_DIR)/samba-admin-api" >&2; exit 1)
	@test -x "$(BIN_DIR)/samba-admin-agent" || (echo "executavel ausente: $(BIN_DIR)/samba-admin-agent" >&2; exit 1)
	@test -f +MANIFEST.in
	@test -f pkg-plist
	@test -f scripts/+PRE_INSTALL
	@test -f scripts/+PRE_DEINSTALL
	@test "$(ARCH)" = "$$($(PKG) config ABI)" || (echo "ABI do package diverge do host: $(ARCH)" >&2; exit 1)
	@printf '%s\n' "$(PACKAGE_ORIGIN)" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._+/-]*$$' || (echo "PACKAGE_ORIGIN invalido" >&2; exit 1)
	@printf '%s\n' "$(PACKAGE_MAINTAINER)" | grep -Eq '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$$' || (echo "PACKAGE_MAINTAINER invalido" >&2; exit 1)
	@printf '%s\n' "$(PACKAGE_WWW)" | grep -Eq '^https://[A-Za-z0-9._~:/?#@!$$&()*+,;=%-]+$$' || (echo "PACKAGE_WWW deve ser uma URL HTTPS valida" >&2; exit 1)
	@if [ "$(RELEASE)" = "yes" ]; then \
		test "$(PACKAGE_ORIGIN)" != "local/samba-admin" || { echo "PACKAGE_ORIGIN institucional e obrigatorio para release" >&2; exit 1; }; \
		case "$(PACKAGE_MAINTAINER)" in *.invalid) echo "PACKAGE_MAINTAINER institucional e obrigatorio para release" >&2; exit 1;; esac; \
		case "$(PACKAGE_WWW)" in *example.invalid*) echo "PACKAGE_WWW institucional e obrigatorio para release" >&2; exit 1;; esac; \
	fi

stage: check safe-stage
	install -d -m 0755 "$(STAGEDIR)$(BINDIR)" "$(STAGEDIR)$(LIBEXECDIR)" "$(STAGEDIR)$(ETCDIR)" "$(STAGEDIR)$(RCDIR)"
	install -m 0555 "$(BIN_DIR)/samba-admin-api" "$(STAGEDIR)$(BINDIR)/samba-admin-api"
	install -m 0555 "$(BIN_DIR)/samba-admin-agent" "$(STAGEDIR)$(BINDIR)/samba-admin-agent"
	install -m 0555 libexec/samba-admin-api-wrapper "$(STAGEDIR)$(LIBEXECDIR)/samba-admin-api-wrapper"
	install -m 0555 libexec/samba-admin-agent-wrapper "$(STAGEDIR)$(LIBEXECDIR)/samba-admin-agent-wrapper"
	install -m 0555 rc.d/samba_admin_api "$(STAGEDIR)$(RCDIR)/samba_admin_api"
	install -m 0555 rc.d/samba_admin_agent "$(STAGEDIR)$(RCDIR)/samba_admin_agent"
	install -m 0640 etc/api.env.sample "$(STAGEDIR)$(ETCDIR)/api.env.sample"
	install -m 0600 etc/agent.env.sample "$(STAGEDIR)$(ETCDIR)/agent.env.sample"
	install -m 0640 etc/shared.env.sample "$(STAGEDIR)$(ETCDIR)/shared.env.sample"
	install -d -m 0755 "$(STAGEDIR)$(PREFIX)/etc/newsyslog.conf.d"
	install -m 0444 etc/newsyslog.conf.d/samba-admin.conf "$(STAGEDIR)$(PREFIX)/etc/newsyslog.conf.d/samba-admin.conf"
	install -m 0444 pkg-descr "$(STAGEDIR)/+DESC"
	install -m 0444 pkg-message "$(STAGEDIR)/+DISPLAY"
	sed -e 's|@VERSION@|$(VERSION)|g' \
	    -e 's|@ARCH@|$(ARCH)|g' \
	    -e 's|@ORIGIN@|$(PACKAGE_ORIGIN)|g' \
	    -e 's|@MAINTAINER@|$(PACKAGE_MAINTAINER)|g' \
	    -e 's|@WWW@|$(PACKAGE_WWW)|g' \
	    +MANIFEST.in > "$(STAGEDIR)/+MANIFEST"
	@printf '\nscripts: {\n  pre-install: <<EOD\n' >> "$(STAGEDIR)/+MANIFEST"
	@sed '1d' scripts/+PRE_INSTALL >> "$(STAGEDIR)/+MANIFEST"
	@printf 'EOD\n  post-install: <<EOD\n' >> "$(STAGEDIR)/+MANIFEST"
	@sed '1d' scripts/+POST_INSTALL >> "$(STAGEDIR)/+MANIFEST"
	@printf 'EOD\n  pre-deinstall: <<EOD\n' >> "$(STAGEDIR)/+MANIFEST"
	@sed '1d' scripts/+PRE_DEINSTALL >> "$(STAGEDIR)/+MANIFEST"
	@printf 'EOD\n  post-deinstall: <<EOD\n' >> "$(STAGEDIR)/+MANIFEST"
	@sed '1d' scripts/+POST_DEINSTALL >> "$(STAGEDIR)/+MANIFEST"
	@printf 'EOD\n}\n' >> "$(STAGEDIR)/+MANIFEST"

package: stage
	@mkdir -p "$(PACKAGE_DIR)"
	$(PKG) create -n -m "$(STAGEDIR)" -p "$(PACKAGE_ROOT)/pkg-plist" -r "$(STAGEDIR)" -o "$(PACKAGE_DIR)"
	sha256 -q "$(PACKAGE_DIR)"/*.pkg > "$(PACKAGE_DIR)/SHA256"
	@echo "pacote criado em $(PACKAGE_DIR)"
