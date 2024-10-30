#!/bin/bash
GO_VERSION="1.22.0"
APP_NAME="PacketSleuth" 
APP_PATH="./cmd/main.go"

REMOVE_GO=false

install_go_ubuntu() {
    echo "Устанавливаем Golang $GO_VERSION для Ubuntu/Debian"
    wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    rm go${GO_VERSION}.linux-amd64.tar.gz
}

install_go_centos() {
    echo "Устанавливаем Golang $GO_VERSION для CentOS/RHEL"
    wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    rm go${GO_VERSION}.linux-amd64.tar.gz
}

install_go_macos() {
    echo "Устанавливаем Golang $GO_VERSION для MacOS"
    brew install go
}

install_go_windows() {
    echo "Устанавливаем Golang $GO_VERSION для Windows"
    choco install golang --version ${GO_VERSION} -y
}

remove_go() {
    echo "Удаляем Golang после сборки..."
    if [ "$(uname)" == "Darwin" ]; then
        brew uninstall go
    elif [ -f /etc/debian_version ] || [ -f /etc/redhat-release ]; then
        sudo rm -rf /usr/local/go
    elif [[ "$(uname -s)" == *"MINGW"* || "$(uname -s)" == *"CYGWIN"* ]]; then
        choco uninstall golang -y
    fi
}

if ! command -v go &> /dev/null; then
    echo "Golang не найден, выполняем установку..."
    REMOVE_GO=true

    if [ "$(uname)" == "Darwin" ]; then
        install_go_macos
    elif [ -f /etc/debian_version ]; then
        install_go_ubuntu
    elif [ -f /etc/redhat-release ]; then
        install_go_centos
    elif [[ "$(uname -s)" == *"MINGW"* || "$(uname -s)" == *"CYGWIN"* ]]; then
        install_go_windows
    else
        echo "Операционная система не поддерживается данным скриптом."
        exit 1
    fi

    if ! command -v go &> /dev/null; then
        echo "Golang не был установлен."
        exit 1
    else
        echo "Golang установлен успешно: $(go version)"
    fi
else
    echo "Golang уже установлен: $(go version)"
fi

if [ ! -f "go.mod" ]; then
    echo "Инициализация Go-модуля..."
    go mod init ${APP_NAME}
fi

echo "Обновление зависимостей..."
go mod tidy

echo "Собираем приложение $APP_NAME из $APP_PATH"

GOOS=$(uname | tr '[:upper:]' '[:lower:]')
GOARCH=$(uname -m)

if [[ "$GOARCH" == "x86_64" ]]; then
    GOARCH="amd64"
elif [[ "$GOARCH" == "aarch64" || "$GOARCH" == "arm64" ]]; then
    GOARCH="arm64"
fi

GOOS=$GOOS GOARCH=$GOARCH go build -o $APP_NAME $APP_PATH

if [ -f "./$APP_NAME" ]; then
    echo "Сборка завершена успешно. Скомпилированный файл: ./$APP_NAME"
else
    echo "Ошибка сборки приложения."
    exit 1
fi


if [ "$REMOVE_GO" = true ]; then
    remove_go
    echo "Golang успешно удален после сборки."
fi
