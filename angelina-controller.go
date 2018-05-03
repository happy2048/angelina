package main
import(
	"controller"
	"os"
)
func main() {
	ctrl,_ := controller.NewController() 
	if ctrl == nil {
		os.Exit(1)
	}
	ctrl.Start()
}
