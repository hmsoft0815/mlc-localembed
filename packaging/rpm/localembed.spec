Name:           localembed
Version:        1.2.0
Release:        1%{?dist}
Summary:        Fast Local Text Embedding Service

License:        MIT
URL:            https://github.com/hmsoft0815/localembed
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.20
Requires:       systemd, logrotate

%description
MLC LocalEmbed is a fast, local text embedding service that provides an Ollama-compatible API.
It uses ONNX runtime for high-performance inference on local hardware.

%prep
%setup -q

%build
# Build all binaries
mkdir -p bin
GOWORK=off go build -o bin/mlcembedder ./cmd/server/main.go
GOWORK=off go build -o bin/localembed-cli ./cmd/cli/main.go
GOWORK=off go build -o bin/localembed-preloader ./cmd/preloader/main.go

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_sysconfdir}/localembed
mkdir -p %{buildroot}%{_unitdir}
mkdir -p %{buildroot}%{_sysconfdir}/logrotate.d
mkdir -p %{buildroot}%{_sharedstatedir}/localembed/mlcembed
mkdir -p %{buildroot}%{_localstatedir}/log/localembed
mkdir -p %{buildroot}%{_libdir}/localembed
mkdir -p %{buildroot}%{_mandir}/man1

# Binaries
install -m 0755 bin/mlcembedder %{buildroot}%{_bindir}/mlcembedder
install -m 0755 bin/localembed-cli %{buildroot}%{_bindir}/localembed-cli
install -m 0755 bin/localembed-preloader %{buildroot}%{_bindir}/localembed-preloader

# Shared library
if [ -f libonnxruntime.so ]; then
    install -m 0755 libonnxruntime.so %{buildroot}%{_libdir}/localembed/libonnxruntime.so
fi

# Config
install -m 0644 packaging/rpm/config.yaml.template %{buildroot}%{_sysconfdir}/localembed/config.yaml

# Systemd
install -m 0644 packaging/systemd/localembed.service %{buildroot}%{_unitdir}/localembed.service

# Logrotate
install -m 0644 packaging/logrotate/localembed.logrotate %{buildroot}%{_sysconfdir}/logrotate.d/localembed

# Man pages
install -m 0644 packaging/man/localembed-server.1 %{buildroot}%{_mandir}/man1/localembed-server.1
install -m 0644 packaging/man/localembed-cli.1 %{buildroot}%{_mandir}/man1/localembed-cli.1

%pre
getent group localembed >/dev/null || groupadd -r localembed
getent passwd localembed >/dev/null || \
    useradd -r -g localembed -d %{_sharedstatedir}/localembed -s /sbin/nologin \
    -c "MLC LocalEmbed Service User" localembed
exit 0

%post
%systemd_post localembed.service

%preun
%systemd_preun localembed.service

%postun
%systemd_postun_with_restart localembed.service

%files
%{_bindir}/mlcembedder
%{_bindir}/localembed-cli
%{_bindir}/localembed-preloader
%{_libdir}/localembed/libonnxruntime.so
%dir %{_sysconfdir}/localembed
%config(noreplace) %{_sysconfdir}/localembed/config.yaml
%{_unitdir}/localembed.service
%{_sysconfdir}/logrotate.d/localembed
%{_mandir}/man1/mlcembedder.1.gz
%{_mandir}/man1/localembed-cli.1.gz
%dir %attr(0750, localembed, localembed) %{_sharedstatedir}/localembed
%dir %attr(0750, localembed, localembed) %{_sharedstatedir}/localembed/mlcembed
%dir %attr(0750, localembed, localembed) %{_localstatedir}/log/localembed

%changelog
* Wed Mar 04 2026 Michael Lechner <m.lechner@example.com> - 1.2.0-1
- Update to 1.2.0
- Added /api/ps and /api/show endpoints
- Improved compatibility with legacy Ollama API
- Added startup banner
- Fixed ONNX runtime loading with absolute paths
- Enhanced error checking for tensor creation
- Added support for command-line flags
