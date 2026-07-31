AC_PREREQ([2.69])
AC_INIT([dev-agent-harness], [0.1.0], [https://github.com/autotaker/kakesu/issues])
AC_CONFIG_SRCDIR([cmd/dev-agent-broker/main.go])
AC_CONFIG_AUX_DIR([build-aux])

AC_PROG_INSTALL
AC_PROG_MKDIR_P
AC_PATH_PROG([GO], [go])
AS_IF([test -z "$GO"], [AC_MSG_ERROR([Go is required])])

AC_ARG_WITH([agent-user],
  [AS_HELP_STRING([--with-agent-user=USER], [untrusted command user @<:@dev-agent@:>@])],
  [agent_user=$withval], [agent_user=dev-agent])
AC_ARG_WITH([runtime-user],
  [AS_HELP_STRING([--with-runtime-user=USER], [trusted Codex runtime user @<:@dev-agent-runtime@:>@])],
  [runtime_user=$withval], [runtime_user=dev-agent-runtime])
AC_ARG_WITH([broker-user],
  [AS_HELP_STRING([--with-broker-user=USER], [broker and approval service user @<:@dev-agent-broker@:>@])],
  [broker_user=$withval], [broker_user=dev-agent-broker])
AC_ARG_WITH([systemd-unit-dir],
  [AS_HELP_STRING([--with-systemd-unit-dir=DIR], [systemd unit directory @<:@PREFIX/lib/systemd/system@:>@])],
  [systemd_unit_dir=$withval], [systemd_unit_dir='${prefix}/lib/systemd/system'])
AC_ARG_WITH([sysusers-dir],
  [AS_HELP_STRING([--with-sysusers-dir=DIR], [systemd-sysusers directory @<:@PREFIX/lib/sysusers.d@:>@])],
  [sysusers_dir=$withval], [sysusers_dir='${prefix}/lib/sysusers.d'])
AC_ARG_WITH([tmpfiles-dir],
  [AS_HELP_STRING([--with-tmpfiles-dir=DIR], [systemd-tmpfiles directory @<:@PREFIX/lib/tmpfiles.d@:>@])],
  [tmpfiles_dir=$withval], [tmpfiles_dir='${prefix}/lib/tmpfiles.d'])
AC_ARG_ENABLE([live-tests],
  [AS_HELP_STRING([--enable-live-tests], [enable tests requiring an isolated Ubuntu host])],
  [enable_live_tests=$enableval], [enable_live_tests=no])

for required_value in "$agent_user" "$runtime_user" "$broker_user" "$systemd_unit_dir" "$sysusers_dir" "$tmpfiles_dir"; do
  AS_IF([test -z "$required_value"], [AC_MSG_ERROR([user and install directory options must not be empty])])
done
for configured_user in "$agent_user" "$runtime_user" "$broker_user"; do
  AS_CASE([$configured_user],
    [@<:@A-Za-z_@:>@*], [],
    [AC_MSG_ERROR([configured user names must start with a letter or underscore])])
  AS_CASE([$configured_user],
    [*@<:@!A-Za-z0-9_-@:>@*], [AC_MSG_ERROR([configured user names contain invalid characters])])
done
AS_CASE([$enable_live_tests], [yes|no], [], [AC_MSG_ERROR([--enable-live-tests accepts only yes or no])])

saved_prefix=$prefix
saved_exec_prefix=$exec_prefix
AS_IF([test "x$prefix" = xNONE], [prefix=$ac_default_prefix])
AS_IF([test "x$exec_prefix" = xNONE], [exec_prefix=$prefix])
eval harness_libexecdir=\"$libexecdir\"
prefix=$saved_prefix
exec_prefix=$saved_exec_prefix

AC_SUBST([agent_user])
AC_SUBST([runtime_user])
AC_SUBST([broker_user])
AC_SUBST([systemd_unit_dir])
AC_SUBST([sysusers_dir])
AC_SUBST([tmpfiles_dir])
AC_SUBST([enable_live_tests])
AC_SUBST([harness_libexecdir])

AC_CONFIG_FILES([
  Makefile
  config/harness.json.example
  deploy/systemd/dev-agent-approval.service
  deploy/systemd/dev-agent-broker.service
  deploy/systemd/dev-agent-egress.service
  deploy/sysusers/dev-agent-harness.conf
  deploy/tmpfiles/dev-agent-harness.conf
])
AC_OUTPUT
