%define debug_package %{nil}
%define pkgname {{cpkg_name}}
%define version {{cpkg_version}}
%define bindir {{cpkg_bindir}}
%define etcdir {{cpkg_etcdir}}
%define release {{cpkg_release}}
%define dist {{cpkg_dist}}
%define manage_conf {{cpkg_manage_conf}}
%define binary {{cpkg_binary}}
%define tarball {{cpkg_tarball}}

Name: %{pkgname}
Version: %{version}
Release: %{release}.%{dist}
Summary: The Choria Server Provisioner
License: Apache-2.0
URL: https://choria.io
Group: System Tools
Packager: R.I.Pienaar <rip@devco.net>
Source0: %{tarball}
BuildRoot: %{_tmppath}/%{pkgname}-%{version}-%{release}-root-%(%{__id_u} -n)

%description
Automatically provisions Choria Servers

%prep
%setup -q

%build

%install
rm -rf %{buildroot}
%{__install} -d -m0755  %{buildroot}/etc/sysconfig
%{__install} -d -m0755  %{buildroot}/etc/init.d
%{__install} -d -m0755  %{buildroot}/etc/logrotate.d
%{__install} -d -m0755  %{buildroot}%{bindir}
%{__install} -d -m0755  %{buildroot}%{etcdir}
%{__install} -d -m0755  %{buildroot}/var/log
%{__install} -d -m0756  %{buildroot}/var/run/%{pkgname}
%{__install} -m0644 dist/choria-provisioner.init %{buildroot}/etc/init.d/%{pkgname}
%{__install} -m0644 dist/choria-provisioner.sysconfig %{buildroot}/etc/sysconfig/%{pkgname}
%{__install} -m0644 dist/choria-provisioner-logrotate %{buildroot}/etc/logrotate.d/%{pkgname}
%{__install} -m0640 dist/choria-provisioner.yaml %{buildroot}%{etcdir}/%{pkgname}.yaml
%{__install} -m0755 %{binary} %{buildroot}%{bindir}/%{pkgname}
touch %{buildroot}/var/log/%{pkgname}.log

%clean
rm -rf %{buildroot}

%post
/sbin/chkconfig --add %{pkgname} || :

%postun
if [ "$1" -ge 1 ]; then
  /sbin/service %{pkgname} condrestart &>/dev/null || :
fi

%preun
if [ "$1" = 0 ] ; then
  /sbin/service %{pkgname} stop > /dev/null 2>&1
  /sbin/chkconfig --del %{pkgname} || :
fi

%files
%attr(640, root, nobody) %config(noreplace)%{etcdir}/%{pkgname}.yaml
%{bindir}/%{pkgname}
/etc/logrotate.d/%{pkgname}
%attr(755, root, root)/etc/init.d/%{pkgname}
%attr(644, root, root) %config(noreplace)/etc/sysconfig/%{pkgname}
%attr(755, nobody, nobody)/var/run/%{pkgname}
%attr(640, nobody, nobody)/var/log/%{pkgname}.log

%changelog
* Tue Aug 07 2018 R.I.Pienaar <rip@devco.net>
- Initial Release
