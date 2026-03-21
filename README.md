# keenetic-routes

Утилита командной строки для управления статическими маршрутами на роутерах Keenetic через NDMS RCI API.

## Возможности

- 📤 **Загрузка маршрутов** из YAML файла на роутер
- 🔎 **Разрешение доменов** в IPv4 и обновление hosts
- 💾 **Резервное копирование** текущих маршрутов в YAML файл
- 🗑️ **Очистка** всех статических маршрутов

## Установка

### Через Go

```bash
go install github.com/vladpi/keenetic-routes@latest
```

### Сборка из исходников

```bash
git clone https://github.com/vladpi/keenetic-routes.git
cd keenetic-routes
make install
```

## Конфигурация

Утилита поддерживает несколько способов настройки подключения к роутеру (в порядке приоритета):

1. **Флаги командной строки** (высший приоритет)
2. **Конфигурационный файл** `~/.config/keenetic-routes/config.yaml`
3. **Переменные окружения**
4. **Файл `.env`** в текущей директории

Значения объединяются по полям: если указан флаг, он заменяет только соответствующее поле, остальные берутся из источников ниже по приоритету.

### Способ 1: Флаги командной строки

```bash
keenetic-routes --host 192.168.100.1:280 --user admin --password your_password upload -f routes.yaml
```

### Способ 2: Конфигурационный файл

Создайте файл `~/.config/keenetic-routes/config.yaml`:

```yaml
host: 192.168.100.1:280
user: admin
password: your_password
```

### Способ 3: Переменные окружения

```bash
export KEENETIC_HOST=192.168.100.1:280
export KEENETIC_USER=admin
export KEENETIC_PASSWORD=your_password
```

### Способ 4: Файл .env

Создайте файл `.env` в текущей директории:

```env
KEENETIC_HOST=192.168.100.1:280
KEENETIC_USER=admin
KEENETIC_PASSWORD=your_password
```

### Интерактивная настройка

Используйте команду `config` для создания конфигурационного файла:

```bash
keenetic-routes config init
```

Пароль вводится без отображения символов в терминале.

## Использование

### Загрузка маршрутов

```bash
keenetic-routes upload -f routes.yaml
```

### Обновление hosts по доменам

```bash
keenetic-routes resolve-domains -f routes.yaml
```

### Резервное копирование маршрутов

```bash
keenetic-routes backup -o backup.yaml
```

### Очистка всех маршрутов

```bash
keenetic-routes clear
```

## Формат файла маршрутов

Файл маршрутов должен быть в формате YAML:

```yaml
routes:
  - comment: "Описание группы маршрутов"
    gateway: "192.168.1.1"  # или interface: "ppp0"
    auto: true
    reject: false
    domains:
      - example.com
      - google.com
    hosts:
      - 8.8.8.8
      - 8.8.4.4
      - 192.168.0.0/16
  
  - comment: "Другая группа"
    interface: "Wireguard1"
    auto: true
    hosts:
      - 142.250.0.0/15
      - 172.217.0.0/16
```

**Параметры группы маршрутов:**

- `comment` (опционально) - комментарий для группы маршрутов
- `gateway` или `interface` (обязательно одно из двух) - шлюз или интерфейс для маршрутов
- `auto` (опционально, по умолчанию `false`) - автоматическое добавление маршрута
- `reject` (опционально, по умолчанию `false`) - отклонение пакетов
- `domains` (опционально) - список доменных имён для резолва в IPv4 (команда `resolve-domains`)
- `hosts` (обязательно) - список IPv4/IPv6 адресов или CIDR подсетей

## Примеры

### Загрузка маршрутов для YouTube через Wireguard

```yaml
routes:
  - comment: YouTube
    interface: Wireguard1
    auto: true
    hosts:
      - 74.125.0.0/16
      - 173.194.0.0/16
      - 172.217.0.0/16
```

```bash
keenetic-routes upload -f youtube-routes.yaml
```

### Резервное копирование перед изменениями

```bash
keenetic-routes backup -o routes-backup-$(date +%Y%m%d).yaml
```

## Требования

- Go 1.25 или выше
- Роутер Keenetic с включенным NDMS RCI API (обычно доступен на порту 280)
- Поддерживаются IPv4/IPv6 адреса и подсети; команда `resolve-domains` добавляет только IPv4 адреса
