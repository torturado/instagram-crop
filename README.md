# Go Instagram Grid Splitter

[![Go Report Card](https://goreportcard.com/badge/github.com/torturado/instagram-crop)](https://goreportcard.com/report/github.com/torturado/instagram-crop)

Una herramienta de línea de comandos escrita en Go para dividir imágenes en múltiples tiles (mosaicos) para crear un efecto de cuadrícula panorámica en perfiles de Instagram. Incluye opciones para manejar el tamaño de la imagen, zonas seguras con relleno o desenfoque, y genera una vista previa del mosaico completo.

## Características

*   Divide una imagen en una cuadrícula de `N` filas por `M` columnas.
*   **Manejo de Dimensiones de Instagram:**
    *   Cada tile se genera con dimensiones finales de 1080x1350px (ratio 4:5).
    *   Incluye "zonas seguras" (`safeZoneW`) para asegurar que el contenido principal (1016px de ancho) sea visible en la vista de perfil de Instagram (que recorta a cuadrado).
*   **Modos de Borde (`edge-mode`):**
    *   `pad`: Rellena las zonas seguras laterales con color blanco.
    *   `blur`: Rellena las zonas seguras laterales con una versión desenfocada del borde de la imagen (usando un simple Box Blur).
*   **Manejo de Tamaño de Imagen de Entrada (`resize-mode`):**
    *   Si la imagen es más pequeña que el contenido total del mosaico:
        *   `resize`: Redimensiona la imagen para que encaje.
        *   `pad`: Rellena la imagen con fondo negro hasta alcanzar el tamaño necesario.
    *   Si la imagen es más grande: Se recorta desde el centro.
*   **Numeración de Tiles Optimizada:** Los tiles se guardan en orden inverso (ej., `tile_3.jpg`, `tile_2.jpg`, `tile_1.jpg`) para facilitar su subida a Instagram en la secuencia correcta.
*   **Vista Previa del Mosaico:** Genera una imagen `stitched_preview.jpg` que muestra todos los tiles unidos, con números superpuestos para indicar el orden de subida.
*   Configurable mediante flags en la línea de comandos.

## Requisitos Previos

*   **Go:** Versión 1.18 o superior (aunque podría funcionar con versiones anteriores, no ha sido probado extensamente). Tu script usa `golang.org/x/image` y `golang.org/x/font`, que se descargarán automáticamente si tienes Go configurado correctamente y los archivos `go.mod` y `go.sum` están presentes.

## Instalación / Construcción

1.  **Clona el repositorio:**
    ```bash
    git clone https://github.com/torturado/instagram-crop.git 
    cd instagram-crop
    ```

2.  **Construye el ejecutable:**
    Puedes compilar el script `test.go` con el siguiente comando. Esto creará un ejecutable llamado `instagram-grid-splitter` (o el nombre que prefieras).
    ```bash
    go build -o instagram-grid-splitter test.go
    ```
    Si prefieres usar el nombre por defecto (`test` en Windows o `test.exe`):
    ```bash
    go build test.go
    ```

## Uso

Ejecuta el script desde la línea de comandos, especificando la imagen de entrada y las opciones deseadas.

```bash
./instagram-grid-splitter -in <ruta_a_tu_imagen> -r <filas> -c <columnas> [opciones]
```
O en Windows:
```bash
.\instagram-grid-splitter.exe -in <ruta_a_tu_imagen> -r <filas> -c <columnas> [opciones]
```

**Flags Disponibles:**

*   `-in PATH`:      Ruta a la imagen de entrada (obligatorio).
*   `-r N`:          Número de filas para dividir (por defecto: 1).
*   `-c N`:          Número de columnas para dividir (por defecto: 1).
*   `-out DIR`:      Directorio de salida para los tiles (por defecto: `./output`).
*   `-edge-mode MODE`: Modo para las zonas seguras:
    *   `pad`: Relleno blanco (por defecto).
    *   `blur`: Desenfoque del borde.
*   `-resize-mode MODE`: Acción si la imagen es más pequeña que la cuadrícula:
    *   `resize`: Redimensionar la imagen (por defecto).
    *   `pad`: Rellenar la imagen.
*   `-interactive`: (No implementado actualmente) Usar prompts interactivos.

**Ejemplos:**

1.  **Crear un mosaico simple de 1x3 (una fila, tres columnas) con relleno en los bordes:**
    ```bash
    ./instagram-grid-splitter -in "mi_foto.jpg" -r 1 -c 3
    ```

2.  **Crear un mosaico de 2x2 con bordes desenfocados y redimensionar la imagen si es pequeña:**
    ```bash
    ./instagram-grid-splitter -in "paisaje.png" -r 2 -c 2 -edge-mode blur -resize-mode resize -out "mosaico_paisaje"
    ```

## Estructura del Proyecto

```
.
├── test.go           # El script principal de Go para dividir imágenes
├── go.mod            # Módulo de Go
├── go.sum            # Checksums de dependencias de Go
├── upload_tiles.py   # Script de Python para subir los tiles a Instagram
├── requirements.txt  # Dependencias de Python para upload_tiles.py
├── .gitignore        # Archivos a ignorar por Git
├── LICENSE           # Licencia del proyecto
└── README.md         # Este archivo
```

## Script de Subida a Instagram (`upload_tiles.py`)

Este repositorio también incluye un script de Python (`upload_tiles.py`) que utiliza la biblioteca `instagrapi` para subir los tiles generados por `test.go` a tu perfil de Instagram.

**Características del Script de Subida:**

*   Carga credenciales de Instagram de forma segura desde variables de entorno (`.env` file).
*   Maneja el inicio de sesión en Instagram y reutiliza sesiones guardadas (`session.json`).
*   Busca los archivos `tile_*.jpg` en el directorio especificado (por defecto `tiles/`).
*   **Orden de Subida:** Ordena los tiles numéricamente para asegurar que se suban en la secuencia correcta para formar el mosaico correctamente en el perfil (ej. `tile_1.jpg`, `tile_2.jpg`, ..., `tile_9.jpg`). **Importante:** El script de Go `test.go` genera los tiles en orden inverso (ej., `tile_9.jpg` primero si es una cuadrícula de 3x3), pero el script de subida `upload_tiles.py` espera y los ordena de `tile_1.jpg` a `tile_N.jpg`. Asegúrate de que la salida de `test.go` y la entrada de `upload_tiles.py` (directorio `tiles/`) sean compatibles con esta numeración.
*   Añade un caption configurable a cada post.
*   Incluye un pequeño delay entre subidas para evitar ser bloqueado por Instagram.

**Configuración del Script de Subida:**

1.  **Python:** Asegúrate de tener Python 3.7 o superior instalado.
2.  **Instalar Dependencias:**
    ```bash
    pip install -r requirements.txt
    ```
3.  **Crear Archivo `.env`:**
    En la raíz del proyecto, crea un archivo llamado `.env` (este archivo está en el `.gitignore`, por lo que no se subirá a GitHub) con tus credenciales de Instagram:
    ```env
    INSTA_USER="tu_usuario_de_instagram"
    INSTA_PASS="tu_contraseña_de_instagram"
    ```
4.  **Directorio de Tiles:** Por defecto, el script buscará los tiles en un directorio llamado `tiles/` en la raíz del proyecto. Asegúrate de que los tiles generados por `test.go` (usando la opción `-out tiles`) estén en esta ubicación.

**Uso del Script de Subida:**

Una vez configurado, ejecuta el script:

```bash
python upload_tiles.py
```

El script iniciará sesión, encontrará los tiles, los ordenará y los subirá a tu perfil de Instagram.

**Nota sobre el orden de los tiles:**
El script `test.go` genera los tiles con números que, cuando se ordenan de forma descendente (ej. `tile_9.jpg`, `tile_8.jpg`, ... `tile_1.jpg`), permiten subirlos "del último al primero" para que aparezcan correctamente en el perfil de Instagram.
El script `upload_tiles.py` actualmente los ordena de forma ascendente (`tile_1.jpg`, `tile_2.jpg`, ...). Esto significa que si `test.go` genera `tile_N.jpg` como el *primer* tile que se debería ver en la esquina superior izquierda de la cuadrícula en Instagram (después de subir todos), y `upload_tiles.py` lo sube como `tile_1.jpg`, el orden será el correcto.

Es crucial que la numeración de los tiles por `test.go` y la lógica de ordenación en `upload_tiles.py` sean consistentes para lograr el efecto deseado en Instagram. El script `test.go` genera los tiles como `tile_NUMEROTOTAL.jpg` ... `tile_1.jpg`. El script `upload_tiles.py` los ordena por número ascendente, así que `tile_1.jpg` se subirá primero, luego `tile_2.jpg`, etc. Esto es el orden correcto para que en el perfil de Instagram se visualicen correctamente, ya que las publicaciones más recientes aparecen primero (más arriba o más a la izquierda).

## Licencia

Este proyecto está bajo la Licencia MIT. Consulta el archivo [LICENSE](https://github.com/torturado/instagram-crop/blob/main/LICENSE) para más detalles.

## Contribuciones

Las contribuciones son bienvenidas. Por favor, abre un *issue* primero para discutir lo que te gustaría cambiar o añade un *Pull Request*.

--- 
