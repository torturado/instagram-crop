import sys
import json
from PIL import Image, ImageTk, ExifTags
import tkinter as tk

# --- Parámetros de proporción del grid ---
# Estos deberían coincidir con los de tu script Go
# (columnas * tile_content_width) / (filas * tile_content_height)
# Ejemplo para 3x3 tiles de 1016x1350 (contenido)
GRID_TARGET_W = 3 * 1016  # totalContentW de Go
GRID_TARGET_H = 3 * 1350  # totalContentH de Go
ASPECT = GRID_TARGET_W / GRID_TARGET_H


def get_corrected_image(filepath):
    """Abre la imagen y la corrige según la orientación EXIF."""
    img = Image.open(filepath)
    try:
        for orientation in ExifTags.TAGS.keys():
            if ExifTags.TAGS[orientation] == 'Orientation':
                break
        
        exif = dict(img._getexif().items())

        if exif[orientation] == 3:
            img = img.rotate(180, expand=True)
        elif exif[orientation] == 6:
            img = img.rotate(270, expand=True)
        elif exif[orientation] == 8:
            img = img.rotate(90, expand=True)
    except (AttributeError, KeyError, IndexError):
        # No EXIF data o no tag de orientación
        pass
    return img

if len(sys.argv) < 4:
    print("Usage: python crop_gui.py <image_path> <rows> <cols>")
    sys.exit(1)
img_path = sys.argv[1]
NUM_ROWS = int(sys.argv[2])
NUM_COLS = int(sys.argv[3])
img = get_corrected_image(img_path) # <--- Imagen corregida aquí
img_w, img_h = img.size

# --- Tkinter setup ---
root = tk.Tk()
root.title("Selecciona el recorte (crop) - Mueve el rectángulo ROJO y pulsa OK")

# Limitar el tamaño máximo de la ventana para que quepa en pantalla
MAX_DISPLAY_W = root.winfo_screenwidth() * 0.8
MAX_DISPLAY_H = root.winfo_screenheight() * 0.8
display_aspect = img_w / img_h

if img_w > MAX_DISPLAY_W or img_h > MAX_DISPLAY_H:
    if img_w / MAX_DISPLAY_W > img_h / MAX_DISPLAY_H:
        display_w = int(MAX_DISPLAY_W)
        display_h = int(display_w / display_aspect)
    else:
        display_h = int(MAX_DISPLAY_H)
        display_w = int(display_h * display_aspect)
    
    # Redimensionar imagen para mostrar (manteniendo aspecto)
    # Las coordenadas del crop se escalarán luego al tamaño original
    display_img = img.resize((display_w, display_h), Image.Resampling.LANCZOS)
    scale_factor = img_w / display_w # Factor para escalar coords de vuelta
else:
    display_img = img
    display_w, display_h = img_w, img_h
    scale_factor = 1.0


canvas = tk.Canvas(root, width=display_w, height=display_h) # Usa tamaño de display
canvas.pack()

tk_img = ImageTk.PhotoImage(display_img)
canvas.create_image(0, 0, anchor=tk.NW, image=tk_img)

# --- Crop rectangle inicial (centrado y con aspecto correcto) ---
def initial_crop_on_display():
    # Calcular dimensiones del crop en la imagen *mostrada*
    if display_w / display_h > ASPECT: # La imagen mostrada es más ancha que el aspect del crop
        crop_h_display = display_h
        crop_w_display = int(display_h * ASPECT)
    else: # La imagen mostrada es más alta (o igual) que el aspect del crop
        crop_w_display = display_w
        crop_h_display = int(display_w / ASPECT)
        
    x0 = (display_w - crop_w_display) // 2
    y0 = (display_h - crop_h_display) // 2
    x1 = x0 + crop_w_display
    y1 = y0 + crop_h_display
    return [x0, y0, x1, y1]

crop_rect_coords_display = initial_crop_on_display()
rect = canvas.create_rectangle(*crop_rect_coords_display, outline="red", width=3, tags="crop_rect")

drag_data = {"x": 0, "y": 0, "item": None, "mode": None}
handle_size = 10 # Tamaño de los cuadraditos para redimensionar

# --- Funciones para arrastrar y redimensionar ---
# (Aquí iría la lógica mejorada para mover y redimensionar, es un poco más larga)
# Por ahora, mantenemos solo el movimiento para probar la orientación y el escalado.

def start_drag(event):
    item = canvas.find_withtag(tk.CURRENT)
    if not item or "crop_rect" not in canvas.gettags(item[0]):
        return
    drag_data["x"] = event.x
    drag_data["y"] = event.y
    drag_data["item"] = item[0]

def drag(event):
    if not drag_data["item"]:
        return
    
    dx = event.x - drag_data["x"]
    dy = event.y - drag_data["y"]
    
    x0, y0, x1, y1 = canvas.coords(drag_data["item"])
    
    new_x0 = x0 + dx
    new_y0 = y0 + dy
    new_x1 = x1 + dx
    new_y1 = y1 + dy

    # Limitar al canvas
    new_x0 = max(0, min(new_x0, display_w - (x1 - x0)))
    new_y0 = max(0, min(new_y0, display_h - (y1 - y0)))
    new_x1 = new_x0 + (x1-x0) # Mantiene el tamaño
    new_y1 = new_y0 + (y1-y0) # Mantiene el tamaño

    canvas.coords(drag_data["item"], new_x0, new_y0, new_x1, new_y1)
    
    # --- Ahora dibuja la cuadrícula ---
    canvas.delete("grid_lines") # Borra la cuadrícula anterior
    
    # Asumimos que R y C (filas y columnas) están disponibles
    # Por ejemplo, pasados como argumentos o definidos globalmente
    num_rows = NUM_ROWS # Ejemplo
    num_cols = NUM_COLS # Ejemplo

    rect_x0, rect_y0, rect_x1, rect_y1 = canvas.coords(drag_data["item"])
    crop_display_w = rect_x1 - rect_x0
    crop_display_h = rect_y1 - rect_y0

    tile_w_display = crop_display_w / num_cols
    tile_h_display = crop_display_h / num_rows

    # Líneas verticales
    for i in range(1, num_cols):
        x = rect_x0 + i * tile_w_display
        canvas.create_line(x, rect_y0, x, rect_y1, fill="blue", width=1, tags="grid_lines")

    # Líneas horizontales
    for i in range(1, num_rows):
        y = rect_y0 + i * tile_h_display
        canvas.create_line(rect_x0, y, rect_x1, y, fill="blue", width=1, tags="grid_lines")

    drag_data["x"] = event.x
    drag_data["y"] = event.y

canvas.tag_bind("crop_rect", "<ButtonPress-1>", start_drag)
canvas.tag_bind("crop_rect", "<B1-Motion>", drag)


def save_and_exit():
    # Coordenadas del rectángulo en la imagen *mostrada*
    x0_display, y0_display, x1_display, y1_display = map(int, canvas.coords(rect))
    
    # Escalar coordenadas de vuelta al tamaño de la imagen original
    x0_orig = int(x0_display * scale_factor)
    y0_orig = int(y0_display * scale_factor)
    w_orig = int((x1_display - x0_display) * scale_factor)
    h_orig = int((y1_display - y0_display) * scale_factor)
    
    crop = {"x": x0_orig, "y": y0_orig, "w": w_orig, "h": h_orig}
    
    # Asegurar que el crop no se salga de la imagen original
    crop["x"] = max(0, crop["x"])
    crop["y"] = max(0, crop["y"])
    if crop["x"] + crop["w"] > img_w:
        crop["w"] = img_w - crop["x"]
    if crop["y"] + crop["h"] > img_h:
        crop["h"] = img_h - crop["y"]

    print(f"Original image size: {img_w}x{img_h}")
    print(f"Display image size: {display_w}x{display_h}")
    print(f"Scale factor: {scale_factor}")
    print(f"Crop on display: x={x0_display}, y={y0_display}, w={x1_display-x0_display}, h={y1_display-y0_display}")
    print(f"Calculated crop for original: {crop}")


    with open("crop_coords.json", "w") as f:
        json.dump(crop, f)
    root.destroy()

btn = tk.Button(root, text="OK", command=save_and_exit)
btn.pack()

root.mainloop()
