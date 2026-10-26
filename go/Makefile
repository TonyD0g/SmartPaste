APP    := build/SmartPaste
BUNDLE := build/SmartPaste.app

.PHONY: build test run app clean

build:
	CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o $(APP) .

test:
	go test ./...

run: build
	./$(APP)

app: build
	rm -rf $(BUNDLE)
	mkdir -p $(BUNDLE)/Contents/MacOS $(BUNDLE)/Contents/Resources
	cp $(APP) $(BUNDLE)/Contents/MacOS/SmartPaste
	cp Info.plist $(BUNDLE)/Contents/Info.plist
	cp ../Resources/SmartPaste.icns $(BUNDLE)/Contents/Resources/SmartPaste.icns
	codesign --force --deep --sign - $(BUNDLE)
	@echo "Built $(BUNDLE)"

clean:
	rm -rf build
