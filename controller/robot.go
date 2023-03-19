package controller

import (
	robot "github.com/go-vgo/robotgo"
)

func MoveMouse(x int, y int) {
	robot.Move(x, y)
}

func GetMousePos() (int, int) {
	return robot.Location()
}

func GetScreenSize() (int, int) {
	return robot.GetScreenSize()
}
