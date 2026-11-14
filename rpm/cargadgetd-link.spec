%global _missing_build_ids_terminate_build 0

Name:           cargadgetd-plugin-link
Version:        0.0.1
Release:        1
Summary:        Cargadgetd phone-screen link app
Group:          Applications/System
License:        MIT
Source0:        %{name}-%{version}.tar.bz2
BuildRequires:  golang
BuildRequires:  gcc

%description
Phone screen mirror plugin. Includes setuid cargadgetd-linkcap and
cargadgetd-linktouch helpers.


%prep
%setup -q


%build
set -eu
export GOTOOLCHAIN=local
export GO111MODULE=on
export GOPROXY=off
export CGO_ENABLED=1
mkdir -p _out
go build -mod=vendor -buildvcs=false -buildmode=c-shared -ldflags "-s -w" \
    -o _out/libcargadgetd_app_link.so .
CGO_ENABLED=0 go build -mod=vendor -buildvcs=false -ldflags "-s -w" \
    -o _out/cargadgetd-linkcap ./cmd/linkcap
CGO_ENABLED=0 go build -mod=vendor -buildvcs=false -ldflags "-s -w" \
    -o _out/cargadgetd-linktouch ./cmd/linktouch


%install
set -eu
rm -rf %{buildroot}
install -d %{buildroot}/usr/lib/cargadgetd/plugins
install -m 755 _out/libcargadgetd_app_link.so %{buildroot}/usr/lib/cargadgetd/plugins/libcargadgetd_app_link.so
install -d %{buildroot}%{_datadir}/cargadgetd/bin
install -m 755 _out/cargadgetd-linkcap %{buildroot}%{_datadir}/cargadgetd/bin/cargadgetd-linkcap
install -m 755 _out/cargadgetd-linktouch %{buildroot}%{_datadir}/cargadgetd/bin/cargadgetd-linktouch


%files
%defattr(-,root,root,-)
%dir /usr/lib/cargadgetd
%dir /usr/lib/cargadgetd/plugins
/usr/lib/cargadgetd/plugins/libcargadgetd_app_link.so
%dir %{_datadir}/cargadgetd
%dir %{_datadir}/cargadgetd/bin
%attr(4755,root,root) %{_datadir}/cargadgetd/bin/cargadgetd-linkcap
%attr(4755,root,root) %{_datadir}/cargadgetd/bin/cargadgetd-linktouch
