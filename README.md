# insight-scheduler

scheduler for deploy insightHD on kubernetes,

source code from https://github.com/kelseyhightower/scheduler

change the schedule policy for deploy insightHd in kubernetes statefulset

## Usage
1. add label  type : insight-statefulset
  template:
    metadata:
      labels:
        ...
        type: insight-statefulset
        ...
2. add env in deploy file
        env:
        - name: ambari_cluster_1_agent_0
          value: master1
        - name: ambari_cluster_1_agent_1
          value: master2
        - name: ambari_cluster_1_agent_2
          value: master3
        - name: ambari_cluster_1_agent_3
          value: slave1
        - name: ambari_cluster_1_agent_4
          value: slave2
        - name: ambari_cluster_1_agent_5
          value: slave3

