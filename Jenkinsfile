pipeline {
    agent any

    environment {
        APP_NAME = "url-shortener"
    }

    stages {
        stage('Checkout Source') {
            steps {
                checkout scm
            }
        }

        stage('Build & Test Artifact') {
            steps {
                sh "docker build -t ${APP_NAME}:${BUILD_NUMBER} ."
            }
        }
    }

    post {
        success {
            echo "Successfully verified code state and packaged container image ${APP_NAME}:${BUILD_NUMBER}!"
        }
        failure {
            echo "Build or verification layers failed inside the Docker engine context. Check the log streams above."
        }
    }
}