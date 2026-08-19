project(
  'gtk-hello-android',
  'c',
  version: '1.0.0',
  meson_version: '>= 1.9.0',
  default_options: ['warning_level=2']
)

gtk_dep = dependency('gtk4')

executable(
  'gtk-hello-android',
  'src/main.c',
  dependencies: gtk_dep,
  install: true,
  android_exe_type: 'application'
)
