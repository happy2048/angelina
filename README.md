
**Angelina**

------------

Angelina: 一款开源的，适用于生物信息学分析的任务调度系统，基于kubernetes,redis,glusterfs和golang构建。

一个简单作业的的例子如下：

	1.一个作业分成了任务1，任务2，任务3，任务4
	2.执行的顺序是: 任务1 --> 任务2，任务3 --> 任务4
	3.任务2和任务3是并行执行

![](http://123.56.3.24/blogimage/image_5976211425.png)

angelina主要就是解决上面的任务执行顺序。

主要特点：
	
	1.每一个任务都是kubernetes中的一个deployment,只要做成相应的容器，无需重复部署任务所需的环境。
	2.支持状态记录，每一个任务运行成功以后都会记录其状态，在整个作业运行过程中，如果有任务运行失败，下次启动该作业，直接从错误任务重新开始运行。
	3.通过redis的订阅发布模式实现任务消息的接收与发送。
	4.通过glusterfs实现任务之间文件共享。
	5.可以把任务进一步拆成子任务。
	6.对运行的任务进行监控，出现错误可重新调度任务。
依赖：
	
	1.kubernetes (>= 1.8)
	2.glusterfs （>= 3.8）
	3.redis (>= 3.0)
	4.go (>= 1.8)
	

**安装**

参考文档： [angelina安装](https://github.com/happy2048/angelina/blob/master/INSTALL.md)

**angelina架构图：**

（1）angelina架构图如下图所示：

![](http://123.56.3.24/blogimage/image_6736054558.png)

（2）说明：

* 一个job有一个angelina controller与之对应，负责整个job的task调度，监控，错误恢复等操作。
* 一个job由很多个task组成，每一个task由一种runner与之对应，每一个runner是由一个deployment构成。
* 由angelina client去启动一个job。
* angelina client与angelina controller之间的通信以及angelina controller与angelina runner之间的通信是依靠redis的订阅发布模式。
* runner之间的文件共享通过glusterfs完成。
* task的运行状态存放在redis当中。

**angelina命令行帮助信息**
	
	[root@683ea81c73f6 biofile]# go run angelina.go -h
	Usage:
	  angelina [OPTIONS]

	Application Options:
	  -v, --version   software version.
	  // 打印版本信息
	  -f, --force     force to run all step of the sample,ignore they are succeed or failed last time.
	  // 重新运行所有task,忽略上次运行的状态
	  -n, --name=     Sample name. 
	  // 指定job名称，这里把一个job也称为一个sample
	  -i, --input=    Input directory,which includes some files  that are important to run the sample.
	  // 指定运行该job所需要的文件的目录，应该把需要的文件放在一个目录下。
	  -o, --output=   Output directory,which is a glusterfs mount point,so that copy files to glusterfs.
	  // glusterfs的入口目录，将在该目录下创建一个job名称的目录，与该job相关的task都在该目录下工作，input directory相关的文件也会复制到该目录下。
	  -t, --template= Pipeline template name,the sample will be running by the pipeline template.
	  // 指定template，angelina会按照这个template预先定义好的流程运行job.
	  -T, --tmp=      A temporary pipeline template file,defines the running steps,the sample will be
			  running by it,can't be used with -t.
      //指定一个临时template，不能和-t一起使用
	  -e, --env=      Pass variable to the pipeline template such as TEST="test",this option can be
			  used many time,eg: -e TEST="test1" -e NAME="test".
	  // 指定该选项以后，该选项给定的环境变量会被应用到template中params域，覆盖其默认值。
	  -c, --config=   configure file,which include the values of -f -n -i -o -t.
	  // 指定配置文件，把-f -n -i -o -t的信息写入到配置文件中，然后不用重复指定这些选项的值。
	  -r, --redis=    Redis server address,can't use localhost:6379 and 127.0.0.1:6379,because they can't
		          be accessed by containers,give another address;if the -r option don't give,you must
			  set the System Environment Variable REDISADDR.
	 // 指定angelina所需要的redis server。不能指定127.0.0.1或者localhost，因为容器无法访问，如果不带该选项，那么必须设置REDISADDR系统环境变量，否则angelina无法运行。
	Other Options:
	  -I, --init=     Angelina configure file,the content of the file will be stored in the redis,and
			  use -g option will generate an angelina template configure file.
	  // 初始化angelina使用，后面跟上初始化文件，文件模板由 -g  init 产生
	  -s, --store=    Give a pipeline template file,and store it to redis.
	  // 生成一个新的template,并且保存
	  -l, --list      List the pipelines which have already existed.
	  // 列出已经存在的template
	  -d, --delete=   Delete the pipeline.
	  // 删除已经存在的template
	  -q, --query=    give the pipeline id or pipeline name to get it's content.
	  // 查询template的详细信息
	  -g, --generate= Three value("conf","pipe","init") can be given,"pipe" is to generate a pipeline
			  template file and you can edit it and use -s to store the pipeline;"conf" is to
			  generate running configure file and you can edit it and use -c option to run the
			  sample;"init" is to generate angelina template configure file,then you can edit
			  it and use -I to init the angelina system.
     // 产生相关的模板文件，-g init产生初始化模板文件，-g pipe产生template模板文件,-g conf产生运行job的配置文件（-c 选项使用）, -g pipe,init,conf三个都产生。 
	Help Options:
	  -h, --help      Show this help message

**angelina模板文件书写**

使用 angelina -g pipe 可以产生一个模板文件，只需要在此基础上填写相应的内容即可，模板如下：

	{
		"pipeline-name": "",  // 模板名称
		"pipeline-description": "", // 模板描述
		"pipeline-content": {
			"refer" : {
				"": "",
				"": ""
			},
			"input": ["",""],
			"params": {
				"": "",
				"": ""
			},
			"step1": {
				"pre-steps": ["",""],
				"container": "",
				"command-name": "",
				"command": ["",""],
				"args":["",""],
				"sub-args": [""]
			},
			"step2": {
				"pre-steps": ["",""],
				"container": "",
				"command-name": "",
				"command": ["",""],
				"args":["",""],
				"sub-args": [""]
			}
		}
	}	


模板说明：
	
	（1） pipeline-name： 模板名称
	（2） pipeline-description： 模板描述
	（3） pipeline-content： 模板内容
	（4） pipeline-content: 主要分为四个域： refer,input,params,以及各个step,每个域都必须表示出来，如果没有数据就留空。
refer域的说明：

	（1）主要在这设置一些任务所需的参考文件，比如参考基因组文件等，下面是个例子：
	
	"refer" : {
		"fasta": "reffa/b37/hg19.fasta",
		"dbsnp138": "refvcf/b37/dbsnp138.vcf"
	}
	（2）这个域所涉及的文件都是只读属性，也就是说你不可以在运行job当中去修改这些文件。
	（3）这个域中的文件路径是一个相对路径，主要是相对于之前我们配置的refer-volume，也就是说，假如我的refer-volume下面放了如下目录：
	
		[root@683ea81c73f6 refer]# ll
		total 17045972
		drwxr-xr-x 3 root root        4096 May  5  2017 annovar_db
		drwxr-xr-x 3 root root        4096 May  5  2017 reffa
		drwxr-xr-x 3 root root        4096 May  5  2017 refvcf
		-rw-r--r-- 1 root root 17455058559 Apr 25 02:39 test.tar.gz
		drwxr-xr-x 2 root root        4096 Apr  7 11:19 yang
	
	     如果我需要reffa/b37/hg19.fasta那么我只需要写reffa/b37/hg19.fasta就行，切记路径要写对，否则运行任务失败。
	（4）如果要在后续的step当中引用该域的一些文件，比如我需要hg19.fasta文件，只需要在step当中写成 “refer@fasta”
		就行。
	（5）如果该域没有内容，那么写成如下格式：
		
		"refer": {}
		
input域说明：
	
	（1） input域主要是对输入文件名称进行转换的，如果不转换，默认是原名复制，下面是一个例子：
	
		"input": [
			"*_R1.fastq.gz ==> test1_R1.fastq",
			"*_R2.fastq.gz ==> test1_R2.fastq",
			"a.txt ==> b.txt"
		]
	（2）上面的例子表达的意思是: a.将input目录当中带有“_R1.fastq.gz”后缀的文件，复制到glusterfs中，并且解压缩成test1_R1.fastq(目前只支持gzip的解压缩)；b.将input目录当中带有“_R2.fastq.gz”后缀的文件，复制到glusterfs中，并且解压缩成test1_R2.fastq；c.将input目录当中的a.txt复制到glusterfs，并且重命名为b.txt
	（3） 该域中input目录下每一个匹配到的文件最多只能一个，例如“*_R1.fastq.gz ==> test1_R1.fastq”中，匹配到“*_R1.fastq.gz”的文件至多只有一个，假设在input目录当中有“test_R1.fastq.gz”和“test1_R1.fastq.gz”，将会报错，因为不知道将哪一个文件转化为"test1_R1.fastq"。
	（4）从input目录下复制的所有文件，将会存放在： glusterfs:data-volume/jobName/step0下 （data-volume是之前我们创建的job数据存放目录,jobName是每一个job的名称）
	
params域的说明：

	（1）params域主要是对step当中的命令的参数进行配置，与直接在step配置参数不同的是，该域的值可以在运行job时动态传入，下面是一个例子：
		"params": {
			"FASTQC": "2",
			"TRIM": "/root/Trimmomatic-0.36/trimmomatic-0.36.jar",
			"TRIMDIR":"/root/Trimmomatic-0.36"
		},
	比如上面的例子的当中，可以在命令行通过“-e  FASTQC=5”动态修改这个值。
	（2）在step当中引用params里面的值，比如在step当中需要使用“/root/Trimmomatic-0.36” 这个值，可以在step中使用“params@TRIMDIR”替换。


step域说明：

	（1） step域是由众多的step组成，并且step编号必须从step1开始，连续不间断，不能重复定义，也就是说不能同时出现多个同样的step编号，下面是一个step例子：
		"step1": {
        	"pre-steps": [],
			"command-name":"fastqc",
        	"container": "registry.vega.com:5000/fastqc:1.0",
        	"command": ["fastqc"],
        	"args":["-o step1@","-f fastq","step0@test1_R1.fastq step0@test1_R2.fastq"],
        	"sub-args": []
		}
		pre-steps: 该step所依赖的step,有多少写多少，没有就写成[].
		command-name: 为该step运行的命令取一个别名。不能留空
		container： 运行该step所需要的容器
		command: 该step所需要运行的命令，数组内容会拼接成字符串。
		args: 命令所需的参数，数组内容会拼接成字符串。
		sub-args: 数组类型，数组的长度代表在该step需要启动多少个这样的容器，来处理不同输入不同输出。
	（2） 下面是一个启动多个相同step的例子：
	
		"step2": {
        	"pre-steps": ["step1"],
			"command-name":"test",
        	"container": "registry.vega.com:5000/test:1.0",
        	"command": ["/bin/bash","/root/test.sh"],
        	"args":["name","30"],
        	"sub-args": ["a.out","b.out"]
		}
	angelina会启动两个registry.vega.com:5000/test:1.0 类型的容器来运行step2，第一个容器运行的命令是：“/bin/bash  /root/test.sh name 30 a.out”,第二个容器运行的命令是“/bin/bash /root/test.sh name 30 b.out”
	启动容器的数量有sub-args数组长度确定。
	（3）如果该step只需要运行一个命令，那么只需要将sub-args留空就行。
	（4）如果在该step当中需要引用pre-steps当中的一些文件，可以使用pre-step的名称+“@”来实现，例如下面：
		"step2": {
        	"pre-steps": ["step1"],
			"command-name":"test",
        	"container": "registry.vega.com:5000/test:1.0",
        	"command": ["/bin/bash","/root/test.sh"],
        	"args":["name","30"，"step1@my.txt","refer@fasta","paramas@TRIMDIR"],
        	"sub-args": ["a.out","b.out"]
		}
		step2用到了step1的my.txt，只需要使用step1@my.txt就行。
	（5） 在step当中用到的所有文件都是使用相对路径。
	（6） step0只能被引用，不能被定义。

**一个简单的模板例子**
	
	
	{
		"refer" : {
			"fasta": "reffa/b37/human_g1k_v37_decoy.fasta"
		},
		"input": ["*_R1.fastq.gz ==> test1_R1.fastq","*_R2.fastq.gz ==> test1_R2.fastq"],
		"params": {
			"FASTQC": "2",
			"TRIM": "/root/Trimmomatic-0.36/trimmomatic-0.36.jar",
			"TRIMDIR":"/root/Trimmomatic-0.36"
		},
		"step1": {
        	"pre-steps": [],
			"command-name":"fastqc",
        	"container": "registry.vega.com:5000/fastqc:1.0",
        	"command": ["fastqc"],
        	"args":["-t params@FASTQC","-o step1@","-f fastq","step0@test1_R1.fastq step0@test1_R2.fastq"],
        	"sub-args": []
		},
		"step2": {
        	"pre-steps": [],
			"command-name": "trimmomatic-0.36.jar",
        	"container": "registry.vega.com:5000/trim:1.0",
        	"command": ["java","-jar","params@TRIM"],
        	"args":["PE -phred33","-threads 2","step0@test1_R1.fastq step0@test1_R2.fastq step2@test1_R1_paired.fastq step2@test1_R1_unpaired.fastq step2@test1_R2_paired.fastq step2@test1_R2_unpaired.fastq","LEADING:3 TRAILING:3 SLIDINGWINDOW:4:15 MINLEN:75","ILLUMINACLIP:params@TRIMDIR/adapters/TruSeq3-PE-2.fa:2:30:10"],
        	"sub-args": []
		},
		"step3": {
			"pre-steps":["step2"],
			"command-name":"bwa mem",
			"container": "registry.vega.com:5000/bwa:1.0",
			"command": ["bwa","mem"],
			"args":["-t 1","-M","-R '@RG\\tID:ST_Test_Yang_329_H7NNYALXX_6\\tSM:ST_Test_Liuhong\\tLB:WBJPE171539-01\\tPU:H7NNYALXX_6\\tPL:illumina\\tCN:thorgene'","refer@fasta"],
			"sub-args":["step2@test1_R1_paired.fastq step2@test1_R2_paired.fastq > step3@test1.sam","step2@test1_R1_paired.fastq step2@test1_R2_paired.fastq > step3@test2.sam"]
		}
	}
模板会转化成如下模板（这个例子中job名为mahui,data-volume会被挂载到容器的/mnt/data,refer-volume会被挂载到容器的/mnt/refer）：

	{
		"step1":{
			"Command":"fastqc   -t 2  "
			"CommandName":"fastqc",
			"Args":"-t 2 -o /mnt/data/mahui/step1/  -f fastq  /mnt/data/mahui/step0/test1_R1.fastq /mnt/data/mahui/step0/test1_R2.fastq ",
			"Container":"registry.vega.com:5000/fastqc:1.0",
			"Prestep":[],
			"SubArgs":[]
		},
		"step2":{
			"Command":"java -jar  /root/Trimmomatic-0.36/trimmomatic-0.36.jar  ",
			"CommandName":"trimmomatic-0.36.jar",
			"Args":"PE -phred33 -threads 2  /mnt/data/mahui/step0/test1_R1.fastq /mnt/data/mahui/step0/test1_R2.fastq /mnt/data/mahui/step2/test1_R1_paired.fastq /mnt/data/mahui/step2/test1_R1_unpaired.fastq /mnt/data/mahui/step2/test1_R2_paired.fastq /mnt/data/mahui/step2/test1_R2_unpaired.fastq  LEADING:3 TRAILING:3 SLIDINGWINDOW:4:15 MINLEN:75 ILLUMINACLIP:/root/Trimmomatic-0.36/adapters/TruSeq3-PE-2.fa:2:30:10",
			"Container":"registry.vega.com:5000/trim:1.0",
			"Prestep":[],
			"SubArgs":[]
		},
		"step3":{
			"Command":"bwa mem",
			"CommandName":"bwa mem",
			"Args":"-t 1 -M -R '@RG\\tID:ST_Test_Yang_329_H7NNYALXX_6\\tSM:ST_Test_Liuhong\\tLB:WBJPE171539-01\\tPU:H7NNYALXX_6\\tPL:illumina\\tCN:thorgene'  /mnt/refer/reffa/b37/human_g1k_v37_decoy.fasta  ",
			"Container":"registry.vega.com:5000/bwa:1.0",
			"Prestep":["step2"],
			"SubArgs":[" /mnt/data/mahui/step2/test1_R1_paired.fastq /mnt/data/mahui/step2/test1_R2_paired.fastq > /mnt/data/mahui/step3/test1.sam "," /mnt/data/mahui/step2/test1_R1_paired.fastq /mnt/data/mahui/step2/test1_R2_paired.fastq > /mnt/data/mahui/step3/test2.sam "]
		}
	}






	
