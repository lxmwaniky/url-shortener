pipeline {
    agent none

    environment {
        PROJECT_ID           = 'cloudcartel'
        REGION               = 'us-central1'
        REPOSITORY_NAME      = 'url-shortener'
        IMAGE_NAME           = 'url-shortener'
        IMAGE_BASE           = "${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPOSITORY_NAME}/${IMAGE_NAME}"
        CR_STAGING           = 'url-shortener-staging'
        CR_PROD              = 'url-shortener-prod'
        DB_CONNECTION_STAGING = 'cloudcartel:us-central1:url-shortener-staging'
        DB_CONNECTION_PROD   = 'cloudcartel:us-central1:url-shortener-prod'
        DB_NAME              = 'url-shortener'
        DB_USER              = 'url-shortener-user'
        DB_PORT              = '5432'
        DB_SSLMODE           = 'disable'
        HEALTH_PATH          = '/health'
        REDIS_DB             = '0'
        REDIS_POOL_SIZE      = '10'
        REDIS_MIN_IDLE_CONNS = '2'
        REDIS_DIAL_TIMEOUT   = '5s'
        REDIS_READ_TIMEOUT   = '3s'
        REDIS_WRITE_TIMEOUT  = '3s'
        CLEANUP_INTERVAL     = '24h'
        REDIS_STAGING_HOST   = 'rational-hawk-38488.upstash.io'
        REDIS_STAGING_PORT   = '6379'
        REDIS_PROD_HOST      = 'deep-rooster-40206.upstash.io'
        REDIS_PROD_PORT      = '6379'
    }

    stages {
        stage('Vet & Test') {
            agent {
                docker {
                    image 'golang:1.26-alpine'
                    args '''-e HOME=/tmp \
                            -v /opt/jenkins-cache/go-mod:/tmp/go-mod-cache \
                            -e GOMODCACHE=/tmp/go-mod-cache \
                            -v /opt/jenkins-cache/go-build:/tmp/go-build-cache \
                            -e GOCACHE=/tmp/go-build-cache'''
                }
            }
            steps {
                sh 'go vet ./...'
                sh 'go test -v ./...'
            }
        }

        stage('Build & Push') {
            agent any
            steps {
                script {
                    env.IMAGE_TAG     = env.GIT_COMMIT.take(7)
                    env.IMAGE_SHA     = "${env.IMAGE_BASE}:sha-${env.IMAGE_TAG}"
                    env.IMAGE_STAGING = "${env.IMAGE_BASE}:staging"
                    env.IMAGE_PROD    = "${env.IMAGE_BASE}:prod"

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
        }

        stage('Deploy Staging') {
            agent any
            steps {
                withCredentials([
                    file(credentialsId: 'gcp-sa-key',              variable: 'GCP_SA_KEY'),
                    string(credentialsId: 'db-staging-password',   variable: 'DB_PASSWORD'),
                    string(credentialsId: 'redis-staging-password', variable: 'REDIS_PASSWORD'),
                    string(credentialsId: 'feistel-seed',          variable: 'FEISTEL_SEED')
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
                            --set-env-vars="ENV=staging,\\
DB_HOST=/cloudsql/${DB_CONNECTION_STAGING},\\
DB_PORT=${DB_PORT},\\
DB_NAME=${DB_NAME},\\
DB_USER=${DB_USER},\\
DB_PASSWORD=\$DB_PASSWORD,\\
DB_SSLMODE=${DB_SSLMODE},\\
REDIS_HOST=${REDIS_STAGING_HOST},\\
REDIS_PORT=${REDIS_STAGING_PORT},\\
REDIS_PASSWORD=\$REDIS_PASSWORD,\\
REDIS_DB=${REDIS_DB},\\
REDIS_POOL_SIZE=${REDIS_POOL_SIZE},\\
REDIS_MIN_IDLE_CONNS=${REDIS_MIN_IDLE_CONNS},\\
REDIS_DIAL_TIMEOUT=${REDIS_DIAL_TIMEOUT},\\
REDIS_READ_TIMEOUT=${REDIS_READ_TIMEOUT},\\
REDIS_WRITE_TIMEOUT=${REDIS_WRITE_TIMEOUT},\\
CLEANUP_INTERVAL=${CLEANUP_INTERVAL},\\
FEISTEL_SEED=\$FEISTEL_SEED"
                    """
                }
            }
        }

        stage('Smoke Tests') {
            agent any
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
            agent any
            steps {
                withCredentials([
                    file(credentialsId: 'gcp-sa-key',              variable: 'GCP_SA_KEY'),
                    string(credentialsId: 'db-prod-password',      variable: 'DB_PASSWORD'),
                    string(credentialsId: 'redis-prod-password',   variable: 'REDIS_PASSWORD'),
                    string(credentialsId: 'feistel-seed',          variable: 'FEISTEL_SEED')
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
                            --set-env-vars="ENV=production,\\
DB_HOST=/cloudsql/${DB_CONNECTION_PROD},\\
DB_PORT=${DB_PORT},\\
DB_NAME=${DB_NAME},\\
DB_USER=${DB_USER},\\
DB_PASSWORD=\$DB_PASSWORD,\\
DB_SSLMODE=${DB_SSLMODE},\\
REDIS_HOST=${REDIS_PROD_HOST},\\
REDIS_PORT=${REDIS_PROD_PORT},\\
REDIS_PASSWORD=\$REDIS_PASSWORD,\\
REDIS_DB=${REDIS_DB},\\
REDIS_POOL_SIZE=${REDIS_POOL_SIZE},\\
REDIS_MIN_IDLE_CONNS=${REDIS_MIN_IDLE_CONNS},\\
REDIS_DIAL_TIMEOUT=${REDIS_DIAL_TIMEOUT},\\
REDIS_READ_TIMEOUT=${REDIS_READ_TIMEOUT},\\
REDIS_WRITE_TIMEOUT=${REDIS_WRITE_TIMEOUT},\\
CLEANUP_INTERVAL=${CLEANUP_INTERVAL},\\
FEISTEL_SEED=\$FEISTEL_SEED"
                    """
                }
            }
        }
    }
}
