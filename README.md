# logcore

Logcore es una pequeña librería de logging en Go que ofrece:
- Salida en texto o JSON.
- Niveles de log: DEBUG, INFO, WARN, ERROR, FATAL.
- Soporte para campos adicionales (Field), capas (layer) y caller.
- Salida hacia múltiples writers (stdout, buffers, archivos, etc).
- Formateo con colores cuando se ejecuta en un terminal.

## Estado
Módulo en desarrollo. Pruebas unitarias disponibles en [`logger/logger_test.go`](logger/logger_test.go).

## Instalación

Clona el repositorio y usa `go build` / `go test` según necesites.

## Uso rápido

Importa el paquete y crea un logger:

```go
package main

import (
    "bytes"
    "fmt"

    "github.com/NeRo0128/logcore/logger"
)

func main() {
    // Crear logger con JSON habilitado y nivel INFO
    l := logger.NewLogger(
        logger.WithLevel(logger.InfoLevel),
        logger.WithJSON(true),
    )

    // Añadir una salida extra (por ejemplo un buffer)
    var buf bytes.Buffer
    l.AddOutput(&buf)

    // Logear un mensaje con campos
    l.Info("Aplicación iniciada", logger.Field{Key: "version", Value: "v0.1.0"})

    fmt.Println(buf.String())
}
```

Funciones útiles:
- Creación: [`logger.NewLogger`](logger/logger.go), [`logger.NewDebugLogger`](logger/logger.go).
- Opciones: [`logger.WithJSON`](logger/options.go), [`logger.WithPrettyJSON`](logger/options.go), [`logger.WithLevel`](logger/options.go), [`logger.WithLayer`](logger/options.go), [`logger.WithField`](logger/options.go), [`logger.WithCaller`](logger/options.go).
- Tipo de campo: [`logger.Field`](logger/logger.go).

También existe un envoltorio simple de uso global en [logs.go](logs.go) (funciones `LogSuccess`, `LogInfo`, `LogWarning`, `LogError`).

## Formato y utilidades
El formateo de JSON y texto se realiza en el paquete interno de utilidades:
- [`utils.FormatJSON`](internal/utils/formatter.go)
- [`utils.FormatText`](internal/utils/formatter.go)

El formateador aplica colores si la salida es un TTY y serializa campos adicionales en el log.

## Ejemplos
- Logging en texto (por defecto):
  - usa `logger.NewLogger()` sin `WithJSON(true)`.
- Logging en JSON:
  - usa `logger.NewLogger(logger.WithJSON(true))`.
- Añadir caller: `logger.WithCaller(true)`.
- Añadir capa (layer): `logger.WithLayer("Repository")`.

## Tests
Ejecuta las pruebas con:
```sh
go test ./...
```
Las pruebas del logger están en [`logger/logger_test.go`](logger/logger_test.go).

## Contribuir
1. Crea una rama feature/fix.
2. Envía PR con tests que cubran cambios.
3. Respeta estilo y mutexes actuales para concurrencia en `logger`.

## Licencia
Sin licencia especificada (añade un archivo LICENSE si quieres compartirlo públicamente).