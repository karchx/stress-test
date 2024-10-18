import json
import matplotlib.pyplot as plt

# Leer datos desde un archivo JSON
with open('stress_test_stats_20241018_144943.json', 'r') as f:
    data = json.load(f)

# Extraer las duraciones del JSON
durations = {
    "Min Duration (s)": float(data["min_duration"].replace('s', '')),
    "Average Duration (s)": float(data["average_duration"].replace('s', '')),
    "Max Duration (s)": float(data["max_duration"].replace('s', ''))
}

# Crear el gráfico de barras
plt.figure(figsize=(8, 6))
plt.bar(durations.keys(), durations.values(), color=['#4CAF50', '#2196F3', '#FF5722'])

# Añadir etiquetas y título
plt.title("Comparación de Duraciones: Mínima, Promedio y Máxima", fontsize=14)
plt.ylabel("Duración (segundos)", fontsize=12)
plt.xlabel("Tipo de Duración", fontsize=12)

# Añadir las etiquetas de los valores en las barras
for i, (key, value) in enumerate(durations.items()):
    plt.text(i, value + 0.001, f"{value:.4f}", ha='center', fontsize=10)

# Mostrar el gráfico
plt.tight_layout()
plt.show()
