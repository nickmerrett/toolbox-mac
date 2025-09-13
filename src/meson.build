# Detect platform first
cpu_family = host_machine.cpu_family()
endian = host_machine.endian()
build_system = host_machine.system()

# Allow forcing macOS mode via environment variable for testing
force_macos_build = get_option('force_macos_build')
if force_macos_build
  build_system = 'darwin'
endif

# Platform-specific build wrapper
if build_system == 'darwin'
  go_build_wrapper_file = files('go-build-wrapper-darwin')
  go_build_wrapper_program = find_program('go-build-wrapper-darwin')
else
  go_build_wrapper_file = files('go-build-wrapper')
  go_build_wrapper_program = find_program('go-build-wrapper')
endif

meson_generate_completions_program = find_program('meson_generate_completions.py')
meson_go_fmt_program = find_program('meson_go_fmt.py')

# Base sources that work on all platforms
sources_common = files(
  'toolbox.go',
  'cmd/completion.go',
  'cmd/enter.go',
  'cmd/help.go',
  'cmd/list.go',
  'cmd/rm.go',
  'cmd/rmi.go',
  'cmd/root.go',
  'cmd/rootDefault.go',
  'cmd/rootMigrationPath.go',
  'cmd/root_test.go',
  'cmd/run.go',
  'cmd/utils.go',
  'pkg/nvidia/nvidia.go',
  'pkg/podman/container.go',
  'pkg/podman/errors.go',
  'pkg/podman/podman.go',
  'pkg/podman/containerInspect_test.go',
  'pkg/shell/shell.go',
  'pkg/shell/shell_test.go',
  'pkg/skopeo/skopeo.go',
  'pkg/utils/arch.go',
  'pkg/utils/errors.go',
  'pkg/utils/fedora.go',
  'pkg/utils/rhel.go',
  'pkg/utils/ubuntu.go',
  'pkg/utils/utils_common.go',
  'pkg/utils/utils_test.go',
  'pkg/version/version.go',
)

# Platform-specific sources
if build_system == 'darwin'
  sources = sources_common + files(
    'cmd/create_darwin.go',
    'cmd/initContainer_darwin.go', 
    'pkg/term/term_darwin.go',
    'pkg/utils/utils_darwin.go',
  )
else
  sources = sources_common + files(
    'cmd/create.go',
    'cmd/initContainer.go',
    'pkg/term/term.go',
    'pkg/term/term_test.go',
    'pkg/utils/libsubid-wrappers.c',
    'pkg/utils/utils.go',
    'pkg/utils/utils_cgo.go',
  )
endif

dynamic_linker = ''

if build_system == 'darwin'
  # macOS uses dyld, not a Linux dynamic linker
  dynamic_linker = '/usr/lib/dyld'
  message('macOS build: using dyld dynamic linker')
else
  # Linux and other Unix systems
  if cpu_family == 'aarch64' and endian == 'little'
    dynamic_linker = '/lib/ld-linux-aarch64.so.1'
  elif cpu_family == 'arm' and endian == 'little'
    dynamic_linker = '/lib/ld-linux-armhf.so.3'
  elif cpu_family == 'loongarch64' and endian == 'little'
    dynamic_linker = '/lib64/ld-linux-loongarch-lp64d.so.1'
  elif cpu_family == 'ppc64' and endian == 'little'
    dynamic_linker = '/lib64/ld64.so.2'
  elif cpu_family == 's390x' and endian == 'big'
    dynamic_linker = '/lib/ld64.so.1'
  elif cpu_family == 'x86' and endian == 'little'
    dynamic_linker = '/lib/ld-linux.so.2'
  elif cpu_family == 'x86_64' and endian == 'little'
    dynamic_linker = '/lib64/ld-linux-x86-64.so.2'
  elif cpu_family == 'riscv64' and endian == 'little'
    dynamic_linker = '/lib/ld-linux-riscv64-lp64d.so.1'
  else
    host_machine_description = cpu_family + ' (' + endian + ' endian)'
    error('Please specify dynamic linker for:', host_machine_description)
  endif
endif

message('Host machine dynamic linker:', dynamic_linker)

toolbox_go = custom_target(
  'toolbox',
  command: [
    go_build_wrapper_program,
    meson.current_source_dir(),
    meson.project_build_root(),
    '@OUTPUT@',
    meson.project_version(),
    cc.cmd_array().get(-1),
    dynamic_linker,
    migration_path_for_coreos_toolbox.to_string(),
  ],
  input: sources,
  install: true,
  install_dir: get_option('bindir'),
  output: 'toolbox',
)

if bashcompletionsdir != ''
  custom_target(
    'bash-completion',
    capture: true,
    command: [
      meson_generate_completions_program,
      meson.current_source_dir(),
      'bash',
    ],
    depends: [toolbox_go],
    install: true,
    install_dir: bashcompletionsdir,
    output: 'toolbox.bash',
)
endif

if fishcompletionsdir != ''
  custom_target(
    'fish-completion',
    capture: true,
    command: [
      meson_generate_completions_program,
      meson.current_source_dir(),
      'fish',
    ],
    depends: [toolbox_go],
    install: true,
    install_dir: fishcompletionsdir,
    output: 'toolbox.fish',
)
endif

custom_target(
  'zsh-completion',
  capture: true,
  command: [
    meson_generate_completions_program,
    meson.current_source_dir(),
    'zsh',
  ],
  depends: [toolbox_go],
  install: true,
  install_dir: zshcompletionsdir,
  output: '_toolbox',
)

if shellcheck.found()
  test('shellcheck src/go-build-wrapper', shellcheck, args: [go_build_wrapper_file])
endif

test('go fmt', meson_go_fmt_program, args: [meson.current_source_dir()])
test('go vet', go, args: ['vet', '-c', '3', './...'], workdir: meson.current_source_dir())
test('go test', go, args: ['test', '-vet', 'off', './...'], workdir: meson.current_source_dir())
