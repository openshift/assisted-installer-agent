pipeline {

  environment {
        AGENT_IMAGE = 'quay.io/assisted_installer_agent'
  }
  agent {
    node {
      label 'host'
    }
  }
  stages {
    stage('build') {
      steps {
        sh 'skipper make build'
      }
    }

    stage('test') {
       steps {
         sh 'skipper make subsystem'
       }
    }

  stage('publish images on push to master') {
              when {
                  branch 'master'
              }

              steps {
                  withCredentials([usernamePassword(credentialsId: 'ocpmetal_cred', passwordVariable: 'PASS', usernameVariable: 'USER')]) {
                      sh '''docker login quay.io -u $USER -p $PASS'''
                  }

                  sh '''docker tag  ${AGENT_IMAGE} ${AGENT_IMAGE}:latest'''
                  sh '''docker tag  ${AGENT_IMAGE} ${AGENT_IMAGE}:${GIT_COMMIT}'''
                  sh '''docker push ${AGENT_IMAGE}:latest'''
                  sh '''docker push ${AGENT_IMAGE}:${GIT_COMMIT}'''
              }
   }
}
}