APP := build/SmartPaste.app

.PHONY: build app run clean

build:
	./Scripts/build-binary.sh

app: build
	rm -rf $(APP)
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources
	cp build/SmartPaste $(APP)/Contents/MacOS/SmartPaste
	cp Info.plist $(APP)/Contents/Info.plist
	cp Resources/SmartPaste.icns Resources/SmartPasteMenuTemplate.png Resources/SmartPasteMenuTemplate@2x.png $(APP)/Contents/Resources/
	codesign --force --deep --sign - $(APP)
	@echo "Built $(APP)"

run: app
	open $(APP)

clean:
	rm -rf build .build
