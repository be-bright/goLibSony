package main

import (
	"fmt"

	"github.com/be-bright/libsonyapi_go/actions"
	"github.com/be-bright/libsonyapi_go/camera"
)

func main() {
	cam := camera.NewCamera()
	camInfo := cam.Info()
	fmt.Println(camInfo)

	fmt.Println(cam.Name)
	fmt.Println(cam.APIVersion)

	cam.Do(actions.Actions().ActTakePicture)

	fNumber := cam.Do(actions.Actions().GetFNumber)
	fmt.Println(fNumber)

	cam.Do(actions.Actions().SetFNumber, "5")
}
