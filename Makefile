.PHONY: clean all deploy

include .env

all: clean surf-vcr_1.0.0_${DEPLOY_ARCH}.deb

clean:
	rm -f *.deb ./surf-vcr

all: surf-vcr

surf-vcr:
	GOOS=${DEPLOY_PLATFORM} GOARCH=${DEPLOY_ARCH} go build -o surf-vcr .

surf-vcr_1.0.0_${DEPLOY_ARCH}.deb: surf-vcr
	nfpm pkg --config surf-vcr.nfpm.yaml --target surf-vcr_1.0.0_${DEPLOY_ARCH}.deb

deploy: surf-vcr_1.0.0_${DEPLOY_ARCH}.deb
	rsync -avzP surf-vcr_1.0.0_${DEPLOY_ARCH}.deb ${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_PATH}
	ssh ${DEPLOY_USER}@${DEPLOY_HOST} "cd ${DEPLOY_PATH} && sudo dpkg -i surf-vcr_1.0.0_${DEPLOY_ARCH}.deb"
