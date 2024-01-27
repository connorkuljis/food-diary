build:
	go build -o main

run: 
	./main

clean:
	rm ./main

all: build run

dev:
	reflex -s make all


