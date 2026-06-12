# Go binaries are statically linked against libtokenizers.a and ship as
# prebuilt artifacts in the source tarball's bin/ directory. Disable the
# debuginfo package — rpm's ELF debug extraction does not play well with
# Go binaries and there are no source-built objects to strip here.
%global debug_package %{nil}

# Fall back to the standard systemd unit dir when the host's rpm install does
# not ship systemd-rpm-macros (e.g. when cross-building on a non-RPM distro).
%{!?_unitdir: %global _unitdir /usr/lib/systemd/system}

# Pin the state dir to /var/lib regardless of build host. Some rpm builds
# (notably on non-RPM distros) default the sharedstatedir macro to the legacy
# /usr/com, which would diverge from the hardcoded /var/lib/localembed paths in
# the systemd unit and config.yaml.template.
%global _sharedstatedir /var/lib

Name:           localembed
Version:        1.5.1
Release:        1%{?dist}
Summary:        Fast Local Text Embedding Service

License:        MIT
URL:            https://github.com/hmsoft0815/localembed
Source0:        %{name}-%{version}.tar.gz

Requires:       systemd, logrotate
Requires(pre):  shadow-utils
# Newer rpm (>=4.19) auto-derives Requires: user(localembed)/group(localembed)
# from the %attr file ownership. We create that user/group in %pre, so declare
# the matching Provides to satisfy those self-references and keep the package
# installable standalone.
Provides:       user(localembed)
Provides:       group(localembed)

%description
MLC LocalEmbed is a fast, local text embedding service that provides an Ollama-compatible API.
It uses ONNX runtime for high-performance inference on local hardware.

%prep
%setup -q

%build
# Binaries are prebuilt (bin/) and bundled in the source tarball. The CGO
# build with the bundled Rust tokenizer static library happens on the build
# host via `task build`; we package the resulting artifacts as-is.
:

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_sysconfdir}/localembed
mkdir -p %{buildroot}%{_unitdir}
mkdir -p %{buildroot}%{_sysconfdir}/logrotate.d
mkdir -p %{buildroot}%{_sharedstatedir}/localembed/models
mkdir -p %{buildroot}%{_localstatedir}/log/localembed
mkdir -p %{buildroot}%{_libdir}/localembed
mkdir -p %{buildroot}%{_mandir}/man1

# Binaries (prebuilt names: mlcembedder, cli, preloader)
install -m 0755 bin/mlcembedder %{buildroot}%{_bindir}/mlcembedder
install -m 0755 bin/cli %{buildroot}%{_bindir}/localembed-cli
install -m 0755 bin/preloader %{buildroot}%{_bindir}/localembed-preloader

# Shared library (ONNX runtime, loaded at runtime via ONNX_PATH)
if [ -f libonnxruntime.so ]; then
    install -m 0755 libonnxruntime.so %{buildroot}%{_libdir}/localembed/libonnxruntime.so
fi

# Config
install -m 0644 packaging/rpm/config.yaml.template %{buildroot}%{_sysconfdir}/localembed/config.yaml

# Systemd
install -m 0644 packaging/systemd/localembed.service %{buildroot}%{_unitdir}/localembed.service

# Logrotate
install -m 0644 packaging/logrotate/localembed.logrotate %{buildroot}%{_sysconfdir}/logrotate.d/localembed

# Man pages (install server page under the actual binary name: mlcembedder)
install -m 0644 packaging/man/localembed-server.1 %{buildroot}%{_mandir}/man1/mlcembedder.1
install -m 0644 packaging/man/localembed-cli.1 %{buildroot}%{_mandir}/man1/localembed-cli.1

%pre
getent group localembed >/dev/null || groupadd -r localembed
getent passwd localembed >/dev/null || \
    useradd -r -g localembed -d %{_sharedstatedir}/localembed -s /sbin/nologin \
    -c "MLC LocalEmbed Service User" localembed
exit 0

# NOTE: systemd scriptlets are written out explicitly rather than via the
# %systemd_* macros so the RPM can be built on any host (incl. non-RPM build
# distros where systemd-rpm-macros is absent and the macros would otherwise be
# baked into the package as literal text).

%post
systemctl daemon-reload >/dev/null 2>&1 || :
if [ $1 -eq 1 ] ; then
    # Initial installation: apply the vendor preset (enabled/disabled).
    systemctl --no-reload preset localembed.service >/dev/null 2>&1 || :
fi

%preun
if [ $1 -eq 0 ] ; then
    # Package removal, not upgrade: stop and disable the service.
    systemctl --no-reload disable --now localembed.service >/dev/null 2>&1 || :
fi

%postun
systemctl daemon-reload >/dev/null 2>&1 || :
if [ $1 -ge 1 ] ; then
    # Upgrade: restart the running service onto the new binaries.
    systemctl try-restart localembed.service >/dev/null 2>&1 || :
fi

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
%dir %attr(0750, localembed, localembed) %{_sharedstatedir}/localembed/models
%dir %attr(0750, localembed, localembed) %{_localstatedir}/log/localembed

%changelog
* Fri Jun 12 2026 Michael Lechner <xsltwonder@gmail.com> - 1.5.1-1
- Patch release covering the RPM packaging fixes (build/install/boot verified)

* Fri Jun 12 2026 Michael Lechner <xsltwonder@gmail.com> - 1.5.0-1
- Package prebuilt binaries (statically linked tokenizer) instead of
  rebuilding inside rpmbuild; no network/CGO toolchain needed at RPM build time
- Align binary names (cli/preloader) and man pages with the build output
- Store models under /var/lib/localembed/models to match config.yaml.template
* Wed Mar 04 2026 Michael Lechner <xsltwonder@gmail.com> - 1.2.0-1
- Update to 1.2.0
- Added /api/ps and /api/show endpoints
- Improved compatibility with legacy Ollama API
- Added startup banner
- Fixed ONNX runtime loading with absolute paths
- Enhanced error checking for tensor creation
- Added support for command-line flags
