pipeline {
    agent {
        docker {
            image 'golang:1.26-alpine'
            args '-v /var/run/docker.sock:/var/run/docker.sock'
        }
    }
    environment {
        GOCACHE = "${WORKSPACE}/.cache/go-build"
        GOPATH  = "${WORKSPACE}/.go"
    }
    stages {
        stage('Linting') {
            steps {
                sh 'go vet ./...'
            }
        }
        stage ('test') {
            steps {
                sh 'go test ./...'
            }
        }
        stage('Compilation') {
            steps {
                sh 'CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api'
            }
            
        }
    }
}