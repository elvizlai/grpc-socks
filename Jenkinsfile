pipeline {
  agent any
  stages {
    stage('') {
      agent {
        docker {
          image 'go'
        }

      }
      steps {
        build 'go build'
      }
    }
  }
}