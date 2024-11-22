package logs

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// Log es la instancia global del logger
var Log *logrus.Logger
var once sync.Once

// Initialize configura el logger compartido
func Initialize() {
	once.Do(func() {
		Log = logrus.New()

		// Crear el directorio de logs si no existe
		if _, err := os.Stat("../../logs"); os.IsNotExist(err) {
			err := os.Mkdir("../../logs", 0755)
			if err != nil {
				panic("No se pudo crear el directorio de logs: " + err.Error())
			}
		}

		// Configurar archivo de salida
		logFile, err := os.OpenFile("../../logs/system.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic("No se pudo abrir el archivo de log: " + err.Error())
		}

		Log.SetOutput(logFile)
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		Log.SetLevel(logrus.InfoLevel)
	})
}
