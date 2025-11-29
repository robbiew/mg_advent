# Linux GLIBC Compatibility Fix

## The Problem

If you're running this software on an older Linux distribution, you might encounter an error like:

```
./advent: /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.34' not found (required by ./advent)
```

This happens because the application was compiled on a newer Linux system with a more recent version of the GNU C Library (glibc) than what's available on your system.

## Solutions

There are several ways to resolve this issue:

### 1. Build from Source on Your System

The most reliable solution is to build the application from source on your own system. This ensures the binary is compatible with your specific glibc version.

```bash
# Clone the repository
git clone https://github.com/robbiew/mg_advent.git
cd mg_advent

# Build for your current system
go build -o advent ./cmd/advent
```

This will create a binary that's compatible with your system's glibc version.

### 2. Use a Container

You can run the application in a container (Docker, Podman) with a compatible glibc version:

```bash
# Create a simple Dockerfile
echo 'FROM golang:latest
WORKDIR /app
COPY . .
RUN go build -o advent ./cmd/advent
ENTRYPOINT ["./advent"]' > Dockerfile

# Build and run the container
docker build -t advent .
docker run -it --rm advent
```

### 3. Use a Compatibility Layer

For Debian/Ubuntu-based systems, you can try using the `glibc-compatibility` package:

```bash
# Install the compatibility package
sudo apt-get update
sudo apt-get install libc6-dev

# Try running the application again
./advent
```

### 4. Request a Build for Older Systems

Contact the maintainers to request a build specifically for older Linux distributions with lower glibc requirements. The build script could be modified to use an older Go version and target an older glibc version.

## Technical Details

The error occurs because:

1. The binary was compiled on a system with glibc 2.34 or newer
2. Your system has an older version of glibc
3. The binary requires symbols from glibc 2.34 that aren't available in your version

The current build script (`scripts/build.sh`) compiles the application with the latest Go version, which may link against newer glibc versions. For maximum compatibility, the maintainers could consider:

1. Using an older Go version for Linux builds (similar to how Go 1.20.14 is used for Windows 7 compatibility)
2. Setting `CGO_ENABLED=0` for Linux builds to create a statically linked binary
3. Building in a container with an older Linux distribution

## Checking Your GLIBC Version

To check your system's glibc version:

```bash
ldd --version
```

## Compatibility Information

This application is built using Go, which generally has good backward compatibility. However, the specific glibc version requirements depend on:

1. The Go version used for compilation
2. Whether CGO is enabled
3. The glibc version on the build system

For maximum compatibility, consider building from source on your own system.