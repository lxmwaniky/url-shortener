pipeline {
    agent none

    environment {
        PROJECT_ID              = 'cloudcartel'
        REGION                  = 'us-central1'
        REPOSITORY_NAME         = 'url-shortener'
        IMAGE_NAME              = 'url-shortener'
        IMAGE_BASE              = "${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/${IMAGE_NAME}"
        CR_STAGING              = 'url-shortener-staging'
        CR_PROD                 = 'url-shortener-prod'
        DB_CONNECTION_STAGING   = 'cloudcartel:us-central1:url-shortener-staging'
        DB_CONNECTION_PROD      = 'cloudcartel:us-central1:url-shortener-prod'
        DB_NAME                 = 'url_shortener'
        DB_USER               = 'url-shortener-user'
        HEALTH_PATH             = '/health'
    }

    stages {
        stage('Vet') {
            agent {
                docker {
                    image 'golang:1.26-alpine'
                    args '-e HOME=/tmp -v /tmp/go-mod-cache:/tmp/go-mod-cache -e GOMODCACHE=/tmp/go-mod-cache'
                }
            }
            steps {
                sh 'go vet ./...'
            }
        }
        stage('Test') {
            agent {
                docker {
                    image 'golang:1.26-alpine'
                    args '-e HOME=/tmp -v /tmp/go-mod-cache:/tmp/go-mod-cache -e GOMODCACHE=/tmp/go-mod-cache'
                }
            }
            steps {
                sh 'go test -v ./...'
            }
        }
        stage('Build & Push') {
            agent {
                docker {
                    image 'google/cloud-sdk:alpine'
                    args '-v /var/run/docker.sock:/var/run/docker.sock'
                }
            }
            steps {
                sh 'apk add --no-cache docker-cli'
                script {
                    env.IMAGE_TAG     = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    env.IMAGE_SHA     = "${env.IMAGE_BASE}:sha-${env.IMAGE_TAG}"
                    env.IMAGE_STAGING = "${env.IMAGE_BASE}:staging"
                    env.IMAGE_PROD    = "${env.IMAGE_BASE}:prod"
                }
                withCredentials([file(credentialsId: 'gcp-sa-key', variable: 'GCP_SA_KEY')]) {
                    sh """
                        gcloud auth activate-service-account --key-file=\$GCP_SA_KEY
                        gcloud auth configure-docker ${REGION}-docker.pkg.dev --quiet
                        docker build -t ${env.IMAGE_SHA} .
                        docker tag ${env.IMAGE_SHA} ${env.IMAGE_STAGING}
                        docker push ${env.IMAGE_SHA}
                        docker push ${env.IMAGE_STAGING}
                    """
                }
            }
        }
        stage('Deploy Staging') {
            agent {
                docker { image 'google/cloud-sdk:alpine' }
            }
            steps {
                withCredentials([
                    file(credentialsId: 'gcp-sa-key',            variable: 'GCP_SA_KEY'),
                    string(credentialsId: 'db-staging-password', variable: 'DB_PASSWORD'),
                    string(credentialsId: 'upstash-staging-url', variable: 'REDIS_URL'),
                    string(credentialsId: 'app-secret-key',      variable: 'APP_SECRET_KEY')
                ]) {
                    sh """
                        gcloud auth activate-service-account --key-file=\$GCP_SA_KEY

                        gcloud run deploy ${CR_STAGING} \\
                            --image=${env.IMAGE_STAGING} \\
                            --platform=managed \\
                            --region=${REGION} \\
                            --project=${PROJECT_ID} \\
                            --allow-unauthenticated \\
                            --add-cloudsql-instances=${DB_CONNECTION_STAGING} \\
                            --set-env-vars="ENV=staging,DB_HOST=/cloudsql/${DB_CONNECTION_STAGING},DB_PORT=5432,DB_NAME=${DB_NAME},DB_USER=${DB_USER},DB_PASSWORD=\$DB_PASSWORD,REDIS_URL=\$REDIS_URL,APP_SECRET_KEY=\$APP_SECRET_KEY"
                    """
                }
            }
        }
        stage('Smoke Tests') {
            agent {
                docker { image 'google/cloud-sdk:alpine' }
            }
            steps {
                withCredentials([file(credentialsId: 'gcp-sa-key', variable: 'GCP_SA_KEY')]) {
                    script {
                        env.STAGING_URL = sh(
                            script: """
                                gcloud auth activate-service-account --key-file=\$GCP_SA_KEY
                                gcloud run services describe ${CR_STAGING} \\
                                    --platform=managed \\
                                    --region=${REGION} \\
                                    --project=${PROJECT_ID} \\
                                    --format='value(status.url)'
                            """,
                            returnStdout: true
                        ).trim()
                    }
                }
                sh """
                    sleep 10

                    STATUS=\$(curl -s -o /dev/null -w '%{http_code}' ${env.STAGING_URL}${HEALTH_PATH})
                    if [ "\$STATUS" != "200" ]; then
                        echo "Health check failed: HTTP \$STATUS"
                        exit 1
                    fi
                    echo "Health check passed"
                """
            }
        }

        stage('Approve Prod') {
            agent none
            steps {
                timeout(time: 24, unit: 'HOURS') {
                    input message: 'Deploy to production?', ok: 'Deploy'
                }
            }
        }

        stage('Deploy Prod') {
            agent {
                docker { image 'google/cloud-sdk:alpine' }
            }
            steps {
                withCredentials([
                    file(credentialsId: 'gcp-sa-key',         variable: 'GCP_SA_KEY'),
                    string(credentialsId: 'db-prod-password', variable: 'DB_PASSWORD'),
                    string(credentialsId: 'upstash-prod-url', variable: 'REDIS_URL'),
                    string(credentialsId: 'app-secret-key',   variable: 'APP_SECRET_KEY')
                ]) {
                    sh """
                        gcloud auth activate-service-account --key-file=\$GCP_SA_KEY
                        gcloud auth configure-docker ${REGION}-docker.pkg.dev --quiet

                        docker pull ${env.IMAGE_SHA}
                        docker tag ${env.IMAGE_SHA} ${env.IMAGE_PROD}
                        docker push ${env.IMAGE_PROD}

                        gcloud run deploy ${CR_PROD} \\
                            --image=${env.IMAGE_PROD} \\
                            --platform=managed \\
                            --region=${REGION} \\
                            --project=${PROJECT_ID} \\
                            --allow-unauthenticated \\
                            --add-cloudsql-instances=${DB_CONNECTION_PROD} \\
                            --set-env-vars="ENV=production,DB_HOST=/cloudsql/${DB_CONNECTION_PROD},DB_PORT=5432,DB_NAME=${DB_NAME},DB_USER=${DB_USER},DB_PASSWORD=\$DB_PASSWORD,REDIS_URL=\$REDIS_URL,APP_SECRET_KEY=\$APP_SECRET_KEY"
                    """
                }
            }
        }
    }
}
