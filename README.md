# Sistema Distribuido

Proyecto realizado por Josi Marin, Emanuel Rodriguez y Daniel Sequeira para el segundo proyecto del curso Sistemas Operativos en segundo semestre del año 2024.

Documentación detallada accesible en https://www.notion.so/Proyecto-2-Sistemas-Operativos-Sistema-Distribuido-14530c143bfc805cb6f0c0abb8f4a881?pvs=4 

## Sistema distribuido

El proyecto consiste en desarrollar un emulador de sistema operativo distribuido que permita la gestión
de recursos y procesos en un entorno distribuido simulado. Los estudiantes crearán un entorno de nodos
interconectados, donde cada nodo representará una instancia básica de un sistema operativo capaz de realizar tareas de procesamiento, comunicación y sincronización con otros nodos. Este sistema debe ser capaz
de distribuir procesos y recursos entre los nodos, asegurando la sincronización y coordinación entre ellos y
la tolerancia a fallos básicos.

#### Objetivos de Aprendizaje

• Aplicar conceptos de sistemas distribuidos para gestionar recursos y procesos en un entorno distribuido.
• Implementar mecanismos de sincronización y comunicación para lograr la coordinación entre los nodos.
• Evaluar la escalabilidad y tolerancia a fallos, permitiendo probar el sistema en diferentes escenarios
de carga y en situaciones de fallo simulado.


### Requerimientos Funcionales

#### Funcionalidad de los Nodos

- Gestión de Procesos: Cada nodo debe poder ejecutar procesos y permitir la asignación dinámica de
  estos a cualquier nodo disponible en la red.
- Gestión de Recursos Compartidos: Los nodos deben administrar el acceso a recursos compartidos
  de manera coordinada, evitando condiciones de carrera y conflictos.
- Sincronización entre Nodos: Implementar mecanismos de sincronización para gestionar el acceso a
  recursos compartidos.
- Manejo de Fallos: Cada nodo debe detectar si otro nodo está inactivo o fuera de servicio y redistribuir
  los procesos asignados a ese nodo.

### Simulación de sistema distribuido

#### Funcionalidad de la Red Distribuida

- Comunicación entre Nodos: Implementar un sistema de comunicación que permita a los nodos enviar
    y recibir mensajes sobre el estado de los recursos y la disponibilidad de procesos.

- Asignación Dinámica de Procesos: El sistema debe permitir que los procesos puedan ser transferidos
    de un nodo a otro si un nodo está sobrecargado o falla.

- Balanceo de Carga: Implementar un algoritmo básico de balanceo de carga para asegurar que los
    procesos se distribuyan de manera uniforme entre los nodos.

- Escalabilidad: El emulador debe permitir agregar nuevos nodos a la red distribuida sin necesidad de
    detener el sistema.

- Tolerancia a Fallos: Si un nodo falla, el sistema debe poder detectar el fallo, informar a los otros nodos
    y redistribuir los procesos en los nodos activos restantes.


### Entregables

Código Fuente: Código del emulador, documentado adecuadamente.

Pruebas y Resultados: Documentación de pruebas en cada caso de uso y sus resultados.

Documentación de Diseño: Explicación de la arquitectura, mecanismos de comunicación, sincronización, y gestión de fallos.
